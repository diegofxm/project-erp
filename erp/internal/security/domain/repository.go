package domain

import (
	"context"

	"github.com/google/uuid"
)

// Repository es el puerto de persistencia del módulo security.
type Repository interface {
	Save(ctx context.Context, u User) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByInviteToken(ctx context.Context, token uuid.UUID) (*User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, name, email string) (*User, error)
	SetPassword(ctx context.Context, id uuid.UUID, hash string) error
	List(ctx context.Context) ([]User, error)

	// Vínculos usuario↔empresa (company_id es UUID puro; FK a company schema se añade cuando exista)
	AddCompany(ctx context.Context, userID, companyID uuid.UUID, role string) error
	ListCompanyIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	HasCompany(ctx context.Context, userID, companyID uuid.UUID) (bool, error)
}

// TokenService es el puerto de firma y verificación de JWT.
type TokenService interface {
	Issue(userID, companyID uuid.UUID) (string, error)
	Verify(raw string) (userID, companyID uuid.UUID, err error)
}
