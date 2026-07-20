package settings

import (
	"time"

	"github.com/google/uuid"
)

const DefaultBrandColor = "#14345C"

type IssuerSettings struct {
	IssuerID   uuid.UUID
	BrandColor string
	UpdatedAt  time.Time
}
