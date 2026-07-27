package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, s Sale) (*Sale, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Sale, error)
	List(ctx context.Context, companyID uuid.UUID) ([]Sale, error)
	UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status SaleStatus) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error
}
