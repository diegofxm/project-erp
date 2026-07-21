package prospects

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

type Prospect struct {
	ID         uuid.UUID
	Name       string
	Email      string
	NIT        string
	HasCedula  bool
	HasRut     bool
	Status     Status
	Notes      string
	ReviewedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
