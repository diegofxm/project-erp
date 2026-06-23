package products

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia del catálogo de productos. Update/Delete
// acotan en el propio SQL (WHERE id = $1 AND issuer_id = $2) — mismo criterio que
// customers.Repository, ver el comentario ahí.
type Repository interface {
	Create(ctx context.Context, p Product) (*Product, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]*Product, error)
	Update(ctx context.Context, issuerID, id uuid.UUID, p Product) (*Product, error)
	Delete(ctx context.Context, issuerID, id uuid.UUID) error
}
