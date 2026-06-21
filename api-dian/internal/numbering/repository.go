package numbering

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia de rangos de numeración.
type Repository interface {
	Create(ctx context.Context, nr NumberingRange) (*NumberingRange, error)
	GetByID(ctx context.Context, id uuid.UUID) (*NumberingRange, error)

	// ClaimNext reclama el siguiente número del rango de forma atómica: bajo concurrencia,
	// dos llamadas simultáneas nunca deben recibir el mismo número ni dejar un hueco.
	// Devuelve ErrRangeExhausted si RangeTo ya fue alcanzado.
	ClaimNext(ctx context.Context, id uuid.UUID) (int64, error)
}
