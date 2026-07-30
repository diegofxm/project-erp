// Package electronic orquesta el ciclo de vida de documentos DIAN.
// Delega la construcción del XML y el envío SOAP a cofacture (librería externa).
// Ver _legacy/apidian/internal/edocuments/ para la implementación de referencia.
package domain

import (
	"time"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
)

// Status del ciclo de vida del documento.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusBuilt     Status = "built"
	StatusSent      Status = "sent"
	StatusAccepted  Status = "accepted"
	StatusRejected  Status = "rejected"
	StatusSendError Status = "send_error"
)

// BillingReferenceInput es la referencia obligatoria de NC/ND/NA al documento que corrigen.
type BillingReferenceInput struct {
	Prefix    string `json:"prefix"`
	Number    string `json:"number"`
	CUFE      string `json:"cufe"`
	IssueDate string `json:"issue_date"`
}

// DiscrepancyResponseInput es el motivo de la nota (catálogo List 22 DIAN).
type DiscrepancyResponseInput struct {
	ReferenceID  string `json:"reference_id"`
	ResponseCode string `json:"response_code"`
	Description  string `json:"description"`
}

// Document es un documento electrónico DIAN (FE, NC, ND, DS, NA) en cualquier etapa.
// Customer/Lines/PaymentMeans son snapshots de solo lectura (ver legacy/apidian-architecture.md).
// Prefix/Number/DocumentKey permanecen vacíos mientras Status == draft — solo se llenan en Confirm.
type Document struct {
	ID               uuid.UUID
	CompanyID        uuid.UUID
	NumberingRangeID uuid.UUID

	DianDocumentTypeCode string // "01" FE, "91" NC, "92" ND, "05" DS, "95" NA
	Prefix               string
	Number               int64
	DocumentKey          string // CUFE/CUDE/CUDS
	IssueDate            time.Time
	IssueTime            string
	CurrencyCode         string

	Customer     cofdom.Party
	Lines        []cofdom.Line
	PaymentMeans []cofdom.PaymentMean
	Totals       cofdom.Totals
	Note         string

	// CustomerID — referencia OPCIONAL de trazabilidad al catálogo de clientes.
	CustomerID *uuid.UUID

	// VendorID — referencia OPCIONAL de trazabilidad al catálogo de proveedores (DS/NA).
	VendorID *uuid.UUID

	// Solo NC/ND/NA
	BillingReference    *BillingReferenceInput
	DiscrepancyResponse *DiscrepancyResponseInput
	NoteTypeCode        string // solo NC

	// Solo DS/NA (roles invertidos: Vendor = tercero, Company = comprador)
	Vendor            *cofdom.Party
	OperationTypeCode string // "10" Residente, "11" No Residente
	WithholdingTaxes  []cofdom.Tax

	QRURL     string
	SignedXML string

	// NCCount/NDCount/NACount: cuántas notas referencian este documento (solo en listados)
	NCCount int
	NDCount int
	NACount int

	Status                Status
	DianTrackID           string
	DianStatusCode        string
	DianStatusDescription string
	DianStatusMessage     string
	ApplicationResponseXML string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// RelatedNote es una vista ligera de NC/ND/NA que referencia este documento.
type RelatedNote struct {
	ID                   uuid.UUID
	DianDocumentTypeCode string
	Prefix               string
	Number               int64
	PayableCents         int64
	Status               Status
	IssueDate            *time.Time
}

// ListFilter filtra listados de documentos.
type ListFilter struct {
	DianDocumentTypeCode string
	Status               Status
	SourceDocumentID     *uuid.UUID
	From                 time.Time
	To                   time.Time
	Search               string
	Limit                int
	Offset               int
}
