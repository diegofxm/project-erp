package documents

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia de documentos.
type Repository interface {
	Create(ctx context.Context, d Document) (*Document, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Document, error)
	UpdateDianStatus(ctx context.Context, id uuid.UUID, status Status, trackID, statusCode, statusDescription, statusMessage string) error
}
