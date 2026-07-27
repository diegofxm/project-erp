package contracts

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia de contratos.
type Repository interface {
	Create(ctx context.Context, in CreateInput) (*Contract, error)
	Get(ctx context.Context, id uuid.UUID) (*Contract, error)
	GetActive(ctx context.Context, employeeID uuid.UUID) (*Contract, error)
	List(ctx context.Context, companyID uuid.UUID) ([]*Contract, error)
	Terminate(ctx context.Context, id uuid.UUID, cause string) error
}
