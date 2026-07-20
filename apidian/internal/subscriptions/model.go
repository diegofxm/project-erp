package subscriptions

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCancelled Status = "cancelled"
	StatusSuspended Status = "suspended"
)

type Subscription struct {
	ID        uuid.UUID
	IssuerID  uuid.UUID
	PlanID    uuid.UUID
	PlanName  string // JOIN — no se guarda en subscriptions
	MaxDocumentsPerMonth *int // JOIN desde plans — nil = ilimitado
	Status    Status
	StartedAt time.Time
	EndsAt    *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
