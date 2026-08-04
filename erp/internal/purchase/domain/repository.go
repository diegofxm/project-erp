package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, o PurchaseOrder) (*PurchaseOrder, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*PurchaseOrder, error)
	List(ctx context.Context, companyID uuid.UUID) ([]PurchaseOrder, error)
	UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status PurchaseStatus) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error
	// SetSupportDocumentID registra qué Documento Soporte se generó a partir de esta orden —
	// evita generar dos veces desde la misma orden (ver electronic CreateFromPurchaseUseCase).
	SetSupportDocumentID(ctx context.Context, companyID, id, documentID uuid.UUID) error
}

// WithholdingRepository gestiona las retenciones aplicadas a una orden de compra.
type WithholdingRepository interface {
	Add(ctx context.Context, w PurchaseWithholding) (*PurchaseWithholding, error)
	ListByPurchase(ctx context.Context, purchaseOrderID uuid.UUID) ([]PurchaseWithholding, error)
	// GetWithholdingSummary agrupa las retenciones aplicadas a un proveedor en un año fiscal —
	// base para emitir el certificado de retención anual (ver accounting IssueWithholdingCertificates).
	GetWithholdingSummary(ctx context.Context, companyID, supplierID uuid.UUID, year int) ([]WithholdingSummary, error)
}

// WithholdingSummary es el total retenido a un proveedor por concepto en un año fiscal.
type WithholdingSummary struct {
	ConceptCode string
	ConceptName string
	Base        float64
	Amount      float64
}
