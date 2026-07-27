package domain

import (
	"time"

	"github.com/google/uuid"
)

type PeriodStatus string

const (
	PeriodOpen   PeriodStatus = "OPEN"
	PeriodClosed PeriodStatus = "CLOSED"
)

type AccountingPeriod struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	Year      int
	Month     int
	Status    PeriodStatus
	OpenedAt  time.Time
	ClosedAt  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
