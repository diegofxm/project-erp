package issuers

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia del dominio de emisores.
type Repository interface {
	Create(ctx context.Context, iss Issuer) (*Issuer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Issuer, error)
	GetByNIT(ctx context.Context, nit string) (*Issuer, error)
}
