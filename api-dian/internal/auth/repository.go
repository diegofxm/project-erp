package auth

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia del dominio de autenticación.
type Repository interface {
	Create(ctx context.Context, u User) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}
