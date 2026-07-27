package documents

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListFilter acota una consulta de listado de documentos — todos los campos son opcionales,
// el valor cero significa "sin ese filtro". Limit/Offset siempre llegan ya normalizados
// (nunca cero/negativos, nunca por encima del máximo permitido) porque Service.ListDocuments
// los normaliza antes de llamar al repositorio — ningún Repository debe confiar en que el
// llamador ya los validó.
//
// SourceDocumentID filtra solo los documentos cuya billing_reference apunta a la factura con
// ese ID — devuelve las NC/ND emitidas sobre una factura dada. El repositorio garantiza que
// la factura de origen también pertenezca al mismo emisor, así que un usuario nunca puede
// enumerar notas sobre facturas ajenas.
type ListFilter struct {
	DianDocumentTypeCode string
	Status               Status
	From, To             time.Time
	Limit, Offset        int
	SourceDocumentID     *uuid.UUID
	// Search filtra por nombre de cliente/proveedor o número de documento (ILIKE).
	Search string
}

// Repository define las operaciones de persistencia de documentos.
type Repository interface {
	// Create persiste un documento NUEVO — hoy siempre llega con Status == StatusDraft (ver
	// Service.CreateInvoiceDraft/CreateCreditNoteDraft/CreateDebitNoteDraft): sin
	// prefix/number/document_key/issue_date/issue_time/qr_url/signed_xml todavía.
	Create(ctx context.Context, d Document) (*Document, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Document, error)
	UpdateDianStatus(ctx context.Context, id uuid.UUID, status Status, trackID, statusCode, statusDescription, statusMessage, applicationResponseXML string) error

	// UpdateDraft reemplaza los campos editables de un borrador (todo lo que no depende de
	// reclamar un número) — solo aplica mientras Status == StatusDraft, devuelve
	// ErrDocumentNotDraft si no.
	UpdateDraft(ctx context.Context, d Document) (*Document, error)

	// Confirm persiste los campos que solo se conocen al confirmar (prefix/number/
	// document_key/issue_date/issue_time/qr_url/signed_xml/status) para un documento que
	// hasta ahora era un borrador — devuelve ErrDocumentNotDraft si ya no lo es (ej. doble
	// confirmación concurrente).
	Confirm(ctx context.Context, d Document) (*Document, error)

	// Delete elimina un borrador — solo aplica mientras Status == StatusDraft, devuelve
	// ErrDocumentNotDraft si no (un documento confirmado es inmutable para siempre).
	Delete(ctx context.Context, id uuid.UUID) error

	// ListByIssuer devuelve los documentos de un emisor que cumplan filter, más recientes
	// primero (por fecha de emisión, luego por creación).
	ListByIssuer(ctx context.Context, issuerID uuid.UUID, filter ListFilter) ([]*Document, error)

	// GetBillingStats devuelve métricas de facturación para el dashboard del emisor.
	GetBillingStats(ctx context.Context, issuerID uuid.UUID) (*BillingStats, error)

	// GetRelatedNotes devuelve todos los documentos que referencian (prefix, number) vía
	// billing_reference — NC/ND para una FE, NA para un DS. Escoped al emisor. Retorna lista
	// vacía (no error) si no hay ninguno.
	GetRelatedNotes(ctx context.Context, issuerID uuid.UUID, prefix string, number int64) ([]RelatedNote, error)

	// GetByDocumentKey busca un documento por su CUFE/CUDS (document_key) dentro del emisor.
	// Retorna ErrDocumentNotFound si no existe.
	GetByDocumentKey(ctx context.Context, issuerID uuid.UUID, key string) (*Document, error)
}
