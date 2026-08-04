package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, p Party) (*Party, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Party, error)
	// GetByIdentification busca sin filtrar por rol — una identificación es única por empresa
	// sin importar si ya es cliente, proveedor, o ambos.
	GetByIdentification(ctx context.Context, companyID uuid.UUID, identTypeCode, identNumber string) (*Party, error)
	List(ctx context.Context, companyID uuid.UUID, role Role) ([]Party, error)
	Update(ctx context.Context, p Party) (*Party, error)
	Delete(ctx context.Context, companyID, id uuid.UUID) error
}
