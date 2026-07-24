package accounts

import "context"

// Repository define las operaciones de persistencia del PUC.
type Repository interface {
	GetByCode(ctx context.Context, code string) (*Account, error)
	GetByID(ctx context.Context, id string) (*Account, error)
	List(ctx context.Context) ([]*Account, error)
}
