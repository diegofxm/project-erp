// Package electronic orquesta el ciclo de vida de documentos DIAN.
// Delega la construcción del XML y el envío SOAP a cofacture (librería externa).
// Ver _legacy/apidian/internal/edocuments/ para la implementación de referencia.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status representa el estado del documento en su ciclo de vida.
type Status string

const (
	StatusDraft     Status = "draft"      // borrador, sin número
	StatusBuilt     Status = "built"      // numerado, XML firmado, pendiente de envío
	StatusSent      Status = "sent"       // enviado a la DIAN, esperando respuesta
	StatusAccepted  Status = "accepted"   // aceptado por la DIAN
	StatusRejected  Status = "rejected"   // rechazado por la DIAN (con motivo)
	StatusSendError Status = "send_error" // error de transporte (red, SOAP)
)

// Document representa un documento electrónico DIAN en cualquier etapa de su ciclo.
type Document struct {
	ID        uuid.UUID
	CompanyID uuid.UUID

	// Tipo y numeración
	DianDocumentTypeCode string // "01" FE, "05" DS, "91" NC, "92" ND, "95" NA, "103" NE
	Prefix               string
	Number               int64
	NumberingRangeID     uuid.UUID

	// Identificadores DIAN
	DocumentKey string // CUFE/CUDE/CUDS
	DianTrackID string // ZipKey (habilitación async)
	QRURL       string

	// Tiempos
	IssueDate time.Time
	IssueTime string

	// Estado
	Status                Status
	DianStatusCode        string
	DianStatusDescription string
	DianStatusMessage     string

	// XML firmado (solo en memoria, no se persiste como texto plano)
	SignedXML string `json:"-"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// SignerPort es el contrato que cofacture implementa para firmar el XML.
type SignerPort interface {
	Sign(xmlDoc []byte, certificate []byte, password string) ([]byte, error)
}

// SenderPort es el contrato que cofacture implementa para enviar a la DIAN.
type SenderPort interface {
	SendSync(zipName string, zipBytes []byte) (*SendResult, error)
	SendTestSet(zipName string, zipBytes []byte, testSetID string) (*SendResult, error)
}

type SendResult struct {
	ZipKey            string
	StatusCode        string
	StatusDescription string
	StatusMessage     string
	IsValid           bool
}
