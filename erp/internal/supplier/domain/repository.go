package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, s Supplier) (*Supplier, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Supplier, error)
	GetByIdentification(ctx context.Context, companyID uuid.UUID, identTypeCode, identNumber string) (*Supplier, error)
	List(ctx context.Context, companyID uuid.UUID) ([]Supplier, error)
	Update(ctx context.Context, s Supplier) (*Supplier, error)
	Delete(ctx context.Context, companyID, id uuid.UUID) error
}
