package documents

import (
	"time"

	"github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
)

// Status es el estado del ciclo de vida de un documento dentro de api-dian — independiente
// del StatusCode que devuelve la propia DIAN (eso queda en DianStatusCode).
type Status string

const (
	// StatusDraft: validado y persistido (customer/lines), pero SIN número reclamado, SIN
	// firmar, SIN enviar — el usuario puede revisar/editar/eliminar libremente. Pasar de aquí
	// a StatusBuilt (vía POST /documents/{id}/confirm) es el único punto donde se "gasta" un
	// número real de la DIAN, justo para no quemar consecutivos en borradores con errores.
	StatusDraft     Status = "draft"
	StatusBuilt     Status = "built"      // construido y firmado, todavía no enviado
	StatusSent      Status = "sent"       // enviado a la DIAN, esperando resultado
	StatusAccepted  Status = "accepted"   // la DIAN lo validó
	StatusRejected  Status = "rejected"   // la DIAN lo rechazó
	StatusSendError Status = "send_error" // error de transporte/infraestructura, no un rechazo DIAN
)

// BillingReferenceInput es la referencia obligatoria de una nota al documento que corrige —
// espejo de domain.BillingReference, con nombre propio porque es lo que recibe el payload de
// emisión, no el tipo de cofacture directamente (evita que un cambio interno de cofacture
// obligue a romper la firma pública de IssueCreditNote/IssueDebitNote).
type BillingReferenceInput struct {
	Prefix    string
	Number    string
	CUFE      string
	IssueDate string
}

// DiscrepancyResponseInput es el motivo de la nota — espejo de domain.DiscrepancyResponse.
type DiscrepancyResponseInput struct {
	ReferenceID  string
	ResponseCode string
	Description  string
}

// Document es un documento electrónico DIAN ya construido — Invoice, CreditNote o DebitNote,
// todos en la MISMA tabla (DianDocumentTypeCode distingue cuál es, igual regla de naming que
// dian_document_types) — no una tabla por tipo.
//
// Customer, Lines y PaymentMeans son snapshots de solo lectura (pass-through): se reciben en
// el payload de emisión y se persisten tal cual, porque eso es lo que la ley exige conservar
// junto al documento — nunca se reconstruyen a partir de customers/products ni de nada que
// pueda cambiar después de emitir (ver docs/api-dian-architecture.md sección 9.21).
// Prefix/Number/DocumentKey/IssueDate/IssueTime/QRURL/SignedXML están vacíos (zero value)
// mientras Status == StatusDraft — recién se llenan en Confirm, nunca antes. Un Number == 0
// o un DocumentKey == "" siempre significa "todavía no confirmado", nunca un dato real (un
// número real nunca es 0, ver numbering.validateRange: RangeFrom > 0).
type Document struct {
	ID                   uuid.UUID
	IssuerID             uuid.UUID
	NumberingRangeID     uuid.UUID
	DianDocumentTypeCode string // "01" Invoice, "91" CreditNote, "92" DebitNote
	Prefix               string
	Number               int64
	DocumentKey          string // CUFE (Invoice) o CUDE (CreditNote/DebitNote)
	IssueDate            time.Time
	IssueTime            string
	CurrencyCode         string

	Customer     domain.Party
	Lines        []domain.Line
	PaymentMeans []domain.PaymentMean
	Totals       domain.Totals
	// Note se guarda desde la creación del borrador (no solo al confirmar) porque
	// cofacture/domain.Invoice.Note la necesita al construir el XML, y eso ya no pasa en el
	// mismo instante que la creación — antes vivía solo en la petición HTTP, nunca persistida.
	Note string

	// CustomerID es una referencia OPCIONAL y de solo trazabilidad a internal/customers — NO
	// es la fuente de verdad del documento (eso sigue siendo Customer, arriba) ni participa en
	// construir el XML. nil si la petición no traía un cliente guardado. Ver
	// docs/api-dian-architecture.md sección 9.21 para el porqué de esta separación.
	CustomerID *uuid.UUID

	// Solo aplican a CreditNote/DebitNote — nil en Invoice.
	BillingReference    *BillingReferenceInput
	DiscrepancyResponse *DiscrepancyResponseInput
	NoteTypeCode        string // CreditNoteTypeCode — solo CreditNote tiene este campo en cofacture

	QRURL     string
	SignedXML string // texto del XML firmado — retención legal, no se recalcula después

	Status                Status
	DianTrackID           string // ZipKey de SendTestSetAsync / identificador de seguimiento
	DianStatusCode        string
	DianStatusDescription string // texto humano de dian.Result — ej. "Set de prueba ... Aceptado"
	DianStatusMessage     string // distinto de StatusDescription: la DIAN los usa para cosas distintas

	CreatedAt time.Time
	UpdatedAt time.Time
}
