package domain

import (
	"context"

	"github.com/google/uuid"
)

type AbsenceRepository interface {
	Create(ctx context.Context, in CreateAbsenceInput) (*Absence, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Absence, error)
	ListByEmployee(ctx context.Context, companyID, employeeID uuid.UUID) ([]*Absence, error)
	ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*Absence, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status AbsenceStatus, notes string) error
}
