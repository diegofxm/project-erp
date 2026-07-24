package periods

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusOpen   Status = "OPEN"
	StatusClosed Status = "CLOSED"
)

// AccountingPeriod representa un mes contable. Solo se puede registrar asientos
// en periodos con status OPEN.
type AccountingPeriod struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	Year      int
	Month     int
	Status    Status
	OpenedAt  time.Time
	ClosedAt  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
