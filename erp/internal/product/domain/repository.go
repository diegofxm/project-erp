package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, p Product) (*Product, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Product, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*Product, error)
	List(ctx context.Context, companyID uuid.UUID) ([]Product, error)
	Update(ctx context.Context, p Product) (*Product, error)
	Delete(ctx context.Context, companyID, id uuid.UUID) error
}
