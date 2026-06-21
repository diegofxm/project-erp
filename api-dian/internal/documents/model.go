package documents

import (
	"time"

	"github.com/diegofxm/ubl21dian/domain"
	"github.com/google/uuid"
)

// Status es el estado del ciclo de vida de un documento dentro de api-dian — independiente
// del StatusCode que devuelve la propia DIAN (eso queda en DianStatusCode).
type Status string

const (
	StatusBuilt     Status = "built"      // construido y firmado, todavía no enviado
	StatusSent      Status = "sent"       // enviado a la DIAN, esperando resultado
	StatusAccepted  Status = "accepted"   // la DIAN lo validó
	StatusRejected  Status = "rejected"   // la DIAN lo rechazó
	StatusSendError Status = "send_error" // error de transporte/infraestructura, no un rechazo DIAN
)

// BillingReferenceInput es la referencia obligatoria de una nota al documento que corrige —
// espejo de domain.BillingReference, con nombre propio porque es lo que recibe el payload de
// emisión, no el tipo de ubl21dian directamente (evita que un cambio interno de ubl21dian
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
// Customer, Lines y PaymentMeans son snapshots de solo lectura (pass-through, sin tablas
// customers/products propias — ver docs/api-dian-architecture.md sección 4.2): se reciben en
// el payload de emisión y se persisten tal cual, porque eso es lo que la ley exige conservar
// junto al documento, no porque api-dian los gestione como entidades propias.
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

	// Solo aplican a CreditNote/DebitNote — nil en Invoice.
	BillingReference    *BillingReferenceInput
	DiscrepancyResponse *DiscrepancyResponseInput
	NoteTypeCode        string // CreditNoteTypeCode — solo CreditNote tiene este campo en ubl21dian

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
