package payslips

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia de liquidaciones.
type Repository interface {
	Create(ctx context.Context, in CreateInput, result Result) (*Payslip, error)
	Get(ctx context.Context, id uuid.UUID) (*Payslip, error)
	List(ctx context.Context, companyID uuid.UUID, year, month int) ([]*Payslip, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	GetSMMLV(ctx context.Context, year int) (int64, error)
	GetARLRate(ctx context.Context, year int, riskClass string) (int, error)
}
