package application

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/audit/domain"
)

type repository interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any) error
	List(ctx context.Context, companyID uuid.UUID, filter domain.ListFilter) ([]*domain.Event, error)
}

type UseCase struct {
	repo repository
	log  *slog.Logger
}

func NewUseCase(repo repository, log *slog.Logger) *UseCase {
	return &UseCase{repo: repo, log: log}
}

// Log registra un evento de auditoría. Nunca devuelve error — falla en silencio
// para no bloquear la operación principal.
func (uc *UseCase) Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any) {
	if err := uc.repo.Log(ctx, companyID, userID, action, resourceType, resourceID, metadata); err != nil {
		uc.log.Error("audit log fallido", "action", action, "error", err)
	}
}

func (uc *UseCase) List(ctx context.Context, companyID uuid.UUID, filter domain.ListFilter) ([]*domain.Event, error) {
	return uc.repo.List(ctx, companyID, filter)
}
