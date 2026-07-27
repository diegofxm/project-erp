package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, c Customer) (*Customer, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Customer, error)
	GetByIdentification(ctx context.Context, companyID uuid.UUID, identTypeCode, identNumber string) (*Customer, error)
	List(ctx context.Context, companyID uuid.UUID) ([]Customer, error)
	Update(ctx context.Context, c Customer) (*Customer, error)
	Delete(ctx context.Context, companyID, id uuid.UUID) error
}
