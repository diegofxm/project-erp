package plans

import (
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID                     uuid.UUID
	Name                   string
	Description            string
	MaxDocumentsPerMonth   *int // nil = ilimitado
	MaxIssuers             int
	PriceCOP               int
	IsActive               bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
