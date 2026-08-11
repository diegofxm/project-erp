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
	// Update corrige tipo/fechas/motivo de una ausencia PENDIENTE -- la validación de estado
	// vive en la aplicación (AbsenceUseCase.Update), igual que Approve/Reject.
	Update(ctx context.Context, id uuid.UUID, in CreateAbsenceInput) (*Absence, error)
	// Delete retira una solicitud PENDIENTE -- una vez aprobada/rechazada queda inmutable.
	Delete(ctx context.Context, id uuid.UUID) error
}
