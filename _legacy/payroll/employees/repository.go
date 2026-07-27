package employees

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia de empleados.
type Repository interface {
	Create(ctx context.Context, in CreateInput) (*Employee, error)
	Get(ctx context.Context, id uuid.UUID) (*Employee, error)
	List(ctx context.Context, companyID uuid.UUID) ([]*Employee, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
}
