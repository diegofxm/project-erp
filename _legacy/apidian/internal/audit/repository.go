package audit

import (
	"context"

	"github.com/google/uuid"
)

// Repository persiste y consulta eventos de auditoría.
type Repository interface {
	Insert(ctx context.Context, e Event) error
	List(ctx context.Context, issuerID uuid.UUID, filter ListFilter) ([]Event, error)
}
