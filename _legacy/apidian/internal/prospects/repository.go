package prospects

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, p Prospect, cedulaPDF, rutPDF []byte) (*Prospect, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Prospect, error)
	List(ctx context.Context) ([]Prospect, error)
	Approve(ctx context.Context, id uuid.UUID) (*Prospect, error)
	Reject(ctx context.Context, id uuid.UUID, notes string) (*Prospect, error)
	GetCedulaPDF(ctx context.Context, id uuid.UUID) ([]byte, error)
	GetRutPDF(ctx context.Context, id uuid.UUID) ([]byte, error)
}
