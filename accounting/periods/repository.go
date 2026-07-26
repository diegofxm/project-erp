package periods

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia de periodos contables.
type Repository interface {
	Create(ctx context.Context, p AccountingPeriod) (*AccountingPeriod, error)
	GetByID(ctx context.Context, id uuid.UUID) (*AccountingPeriod, error)
	GetByYearMonth(ctx context.Context, companyID uuid.UUID, year, month int) (*AccountingPeriod, error)
	List(ctx context.Context, companyID uuid.UUID) ([]*AccountingPeriod, error)
	Close(ctx context.Context, id uuid.UUID) error
	// CloseAllForYear cierra todos los periodos OPEN de un año para una empresa.
	CloseAllForYear(ctx context.Context, companyID uuid.UUID, year int) error
}
