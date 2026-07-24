package journals

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia de asientos.
type Repository interface {
	Create(ctx context.Context, entry JournalEntry) (*JournalEntry, error)
	GetByID(ctx context.Context, id uuid.UUID) (*JournalEntry, error)
	Void(ctx context.Context, id uuid.UUID) error
	ListByCompany(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*JournalEntry, error)
}
