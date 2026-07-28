package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	GetBillingStats(ctx context.Context, companyID uuid.UUID) (*BillingStats, error)
}
