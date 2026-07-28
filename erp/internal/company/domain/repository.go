package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, c Company) (*Company, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Company, error)
	GetByNIT(ctx context.Context, nit string) (*Company, error)
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]Company, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]Company, error)
	UpdateProfile(ctx context.Context, c Company) (*Company, error)
	UpdateCredentials(ctx context.Context, id uuid.UUID, softwareID, softwarePIN string, cert []byte, certPwd, neSoftwareID, neSoftwarePIN string) error
	UpdateLogo(ctx context.Context, id uuid.UUID, logo []byte, contentType string) error
	DeleteLogo(ctx context.Context, id uuid.UUID) error
}
