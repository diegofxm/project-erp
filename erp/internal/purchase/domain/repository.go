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
}
