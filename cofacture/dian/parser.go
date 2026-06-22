// Package dian interpreta las respuestas de validación de la DIAN (soap.DianResponse) y las
// convierte en algo que el resto del pipeline pueda usar directamente.
//
// Verificado contra una respuesta real de GetStatusZip: el campo XmlBase64Bytes viene con
// doble codificación base64 — encoding/xml ya decodifica la primera capa (es un campo
// xs:base64Binary), pero el contenido resultante es a su vez texto base64 que hay que
// decodificar otra vez para llegar al ApplicationResponse real. XmlBytes, cuando la DIAN lo
// usa en vez de XmlBase64Bytes, no debería necesitar esa segunda vuelta.
package dian

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/soap"
)

// ValidatorID es el literal fijo que usa la DIAN como identificador del validador,
// confirmado tanto en el ejemplo del anexo técnico como en una respuesta real.
const ValidatorID = "Unidad Especial Dirección de Impuestos y Aduanas Nacionales"

// Message es un mensaje de validación ya separado en sus partes. La DIAN los entrega como
// una sola cadena con el formato "Regla: <código>, <Rechazo|Notificación>: <texto>" — Severity
// queda vacío si el mensaje no sigue ese patrón (no se descarta, Raw siempre tiene el original).
type Message struct {
	Rule     string
	Severity string
	Text     string
	Raw      string
}

// IsRejection es true cuando Severity es "Rechazo" — distingue un rechazo real de una
// simple notificación informativa (la DIAN puede aprobar un documento con notificaciones).
func (m Message) IsRejection() bool {
	return m.Severity == "Rechazo"
}

func parseMessage(raw string) Message {
	msg := Message{Raw: raw}
	rest, ok := strings.CutPrefix(raw, "Regla: ")
	if !ok {
		return msg
	}
	rule, rest, ok := strings.Cut(rest, ", ")
	if !ok {
		return msg
	}
	severity, text, ok := strings.Cut(rest, ": ")
	if !ok {
		return msg
	}
	msg.Rule = rule
	msg.Severity = severity
	msg.Text = text
	return msg
}

// Result es la interpretación de un soap.DianResponse: mensajes ya separados y, si vino, el
// ApplicationResponse embebido ya decodificado.
type Result struct {
	IsValid           bool
	StatusCode        string
	StatusDescription string
	StatusMessage     string
	Messages          []Message

	// XmlDocumentKey es el identificador del documento al que aplica esta respuesta (el
	// CUFE/CUDE de la factura/nota validada), no el del ApplicationResponse embebido.
	XmlDocumentKey string
	XmlFileName    string

	// ApplicationResponseXML es el XML decodificado del ApplicationResponse que emitió la
	// DIAN al validar, listo para usarse como domain.ValidationResult.ApplicationResponseXML.
	// nil si la respuesta no traía ninguno.
	ApplicationResponseXML []byte
}

// HasRejections es true si algún mensaje es un rechazo real (no solo una notificación).
func (r Result) HasRejections() bool {
	for _, m := range r.Messages {
		if m.IsRejection() {
			return true
		}
	}
	return false
}

// Interpret convierte un soap.DianResponse (un elemento de los que devuelve GetStatusZip, o
// el resultado de GetStatus) en un Result.
func Interpret(resp soap.DianResponse) (Result, error) {
	result := Result{
		IsValid:           resp.IsValid,
		StatusCode:        resp.StatusCode,
		StatusDescription: resp.StatusDescription,
		StatusMessage:     resp.StatusMessage,
		XmlDocumentKey:    resp.XmlDocumentKey,
		XmlFileName:       resp.XmlFileName,
	}

	if resp.ErrorMessage != nil {
		for _, raw := range resp.ErrorMessage.Items {
			result.Messages = append(result.Messages, parseMessage(raw))
		}
	}

	embedded, err := decodeEmbeddedXML(resp)
	if err != nil {
		return Result{}, fmt.Errorf("dian: decodificar XML embebido: %w", err)
	}
	result.ApplicationResponseXML = embedded

	return result, nil
}

func decodeEmbeddedXML(resp soap.DianResponse) ([]byte, error) {
	if len(resp.XmlBase64Bytes) > 0 {
		decoded, err := base64.StdEncoding.DecodeString(string(resp.XmlBase64Bytes))
		if err != nil {
			return nil, fmt.Errorf("decodificar la segunda capa de XmlBase64Bytes: %w", err)
		}
		return decoded, nil
	}
	if len(resp.XmlBytes) > 0 {
		return resp.XmlBytes, nil
	}
	return nil, nil
}

// ToValidationResult construye un domain.ValidationResult listo para
// builder.BuildInvoiceAttachedDocument. La DIAN no repite en su respuesta el ID/fecha del
// documento ni el momento en que se hizo la consulta, así que quien llama los aporta.
func (r Result) ToValidationResult(lineID, documentID, documentHashType, documentIssueDate, validationDate, validationTime string) domain.ValidationResult {
	return domain.ValidationResult{
		LineID:                 lineID,
		DocumentID:             documentID,
		DocumentCUFE:           r.XmlDocumentKey,
		DocumentHashType:       documentHashType,
		DocumentIssueDate:      documentIssueDate,
		ApplicationResponseXML: string(r.ApplicationResponseXML),
		ValidatorID:            ValidatorID,
		ValidationResultCode:   r.StatusCode,
		ValidationDate:         validationDate,
		ValidationTime:         validationTime,
	}
}
