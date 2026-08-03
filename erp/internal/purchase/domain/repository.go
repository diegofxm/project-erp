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
