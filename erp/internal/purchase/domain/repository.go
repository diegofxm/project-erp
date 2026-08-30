package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, o PurchaseOrder) (*PurchaseOrder, error)
	// Update reemplaza proveedor/fecha/notas/líneas de una orden EN BORRADOR -- devuelve
	// ErrPurchaseNotDraft si ya está confirmada/recibida (se reversa con Cancel, no se edita).
	Update(ctx context.Context, companyID, id uuid.UUID, o PurchaseOrder) (*PurchaseOrder, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*PurchaseOrder, error)
	List(ctx context.Context, companyID uuid.UUID) ([]PurchaseOrder, error)
	UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status PurchaseStatus) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error
	// SetSupportDocumentID registra qué Documento Soporte se generó a partir de esta orden —
	// evita generar dos veces desde la misma orden (ver electronic CreateFromPurchaseUseCase).
	SetSupportDocumentID(ctx context.Context, companyID, id, documentID uuid.UUID) error
	// NextPurchaseNumber devuelve el siguiente consecutivo de orden de compra para la empresa y
	// el año dados (arranca en 1 cada año). Mismo patrón que sales.Repository.NextSaleNumber.
	NextPurchaseNumber(ctx context.Context, companyID uuid.UUID, year int) (int, error)
	// SetPurchaseNumberCounter fija el consecutivo para que la próxima orden emitida sea
	// nextNumber — solo tiene efecto si nextNumber es mayor al último ya asignado (devuelve
	// ErrNumberCounterBackwards si no). Pensado para migrar una empresa que ya traía su propia
	// numeración de otro sistema.
	SetPurchaseNumberCounter(ctx context.Context, companyID uuid.UUID, year, nextNumber int) (int, error)
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
