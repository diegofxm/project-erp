package documents

import (
	"context"
	"fmt"
	"html"

	"github.com/diegofxm/apidian/internal/email"
	"github.com/diegofxm/apidian/internal/issuers"
	cofzip "github.com/diegofxm/cofacture/zip"
	"github.com/google/uuid"
)

// SendDocumentEmail envía el documento ya aceptado por la DIAN al correo del cliente, con el
// PDF y el XML firmado empacados en un único archivo ZIP — práctica estándar en Colombia para
// entrega de documentos electrónicos al adquiriente.
//
// Válido para Factura (01), Nota Crédito (91) y Nota Débito (92). Solo StatusAccepted: nunca
// un borrador, nunca uno rechazado/con error de envío, nunca StatusSent.
func (s *Service) SendDocumentEmail(ctx context.Context, issuerID, id uuid.UUID) error {
	d, iss, err := s.loadDocumentAndIssuer(ctx, issuerID, id)
	if err != nil {
		return err
	}
	if d.Status != StatusAccepted {
		return ErrDocumentNotAccepted
	}
	if d.Customer.Email == "" {
		return ErrCustomerEmailMissing
	}

	pdfBytes, err := s.RenderDocumentPDF(ctx, issuerID, id)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s%d", d.Prefix, d.Number)
	zipBytes, err := cofzip.Build([]cofzip.File{
		{Name: filename + ".xml", Content: []byte(d.SignedXML)},
		{Name: filename + ".pdf", Content: pdfBytes},
	})
	if err != nil {
		return fmt.Errorf("empacar adjuntos: %w", err)
	}

	typeName := documentTypeName(d.DianDocumentTypeCode)
	// Asunto según Anexo Técnico 1.9 sección 9.1:
	// NIT; Nombre del facturador; Número del documento; Código tipo; Nombre comercial
	tradeName := iss.TradeName
	if tradeName == "" {
		tradeName = iss.BusinessName
	}
	subject := fmt.Sprintf("%s;%s;%s;%s;%s", iss.NIT, iss.BusinessName, filename, d.DianDocumentTypeCode, tradeName)
	msg := email.Message{
		To:      d.Customer.Email,
		Subject: subject,
		BodyText: documentEmailText(d, iss, typeName),
		BodyHTML: documentEmailHTML(d, iss, typeName),
		Attachments: []email.Attachment{
			{Filename: filename + ".zip", ContentType: "application/zip", Content: zipBytes},
		},
	}
	return s.email.Send(ctx, msg)
}

// documentTypeName devuelve el nombre en minúsculas del tipo de documento para usar en emails.
func documentTypeName(typeCode string) string {
	switch typeCode {
	case creditNoteDianDocumentType:
		return "nota crédito"
	case debitNoteDianDocumentType:
		return "nota débito"
	default:
		return "factura electrónica"
	}
}

func documentEmailText(d *Document, iss *issuers.Issuer, typeName string) string {
	return fmt.Sprintf(
		"Hola %s,\n\n"+
			"Adjuntamos tu %s No. %s%d, emitida por %s.\n\n"+
			"El archivo adjunto (ZIP) contiene:\n"+
			"- El XML: el documento electrónico firmado, válido ante la DIAN.\n"+
			"- El PDF: representación gráfica del documento.\n\n"+
			"Este es un mensaje automático, por favor no respondas a esta dirección.\n",
		d.Customer.Name, typeName, d.Prefix, d.Number, iss.BusinessName,
	)
}

// documentEmailHTML escapa los campos interpolados (Customer.Name/iss.BusinessName son datos
// del cliente/snapshot, no se confía en que no traigan caracteres HTML especiales).
func documentEmailHTML(d *Document, iss *issuers.Issuer, typeName string) string {
	return fmt.Sprintf(
		"<p>Hola %s,</p>"+
			"<p>Adjuntamos tu %s No. %s%d, emitida por %s.</p>"+
			"<p>El archivo adjunto (ZIP) contiene:</p>"+
			"<ul><li>El XML: el documento electrónico firmado, válido ante la DIAN.</li>"+
			"<li>El PDF: representación gráfica del documento.</li></ul>"+
			"<p>Este es un mensaje automático, por favor no respondas a esta dirección.</p>",
		html.EscapeString(d.Customer.Name), typeName, html.EscapeString(d.Prefix), d.Number, html.EscapeString(iss.BusinessName),
	)
}
