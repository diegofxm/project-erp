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

// UserInfo es la vista mínima de un usuario que audit necesita para mostrar quién hizo cada
// evento -- ver UserLookupPort.
type UserInfo struct {
	ID    uuid.UUID
	Email string
	Name  string
}

// UserLookupPort resuelve email/nombre para un lote de userIDs -- puerto local hacia security,
// implementado por audit/infrastructure/security.Adapter. Antes, el propio repositorio de audit
// hacía `LEFT JOIN security.users` directo desde SQL, violando la frontera de schema entre
// módulos (ver auditoría 2026-08-09, Fase 2 punto 15). Ahora el repositorio de audit solo conoce
// audit.events; el enriquecimiento con datos de usuario pasa por acá, en la capa de aplicación.
type UserLookupPort interface {
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]UserInfo, error)
}

type UseCase struct {
	repo  repository
	log   *slog.Logger
	users UserLookupPort
}

func NewUseCase(repo repository, log *slog.Logger, users UserLookupPort) *UseCase {
	return &UseCase{repo: repo, log: log, users: users}
}

// Log registra un evento de auditoría. Nunca devuelve error — falla en silencio
// para no bloquear la operación principal.
func (uc *UseCase) Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any) {
	if err := uc.repo.Log(ctx, companyID, userID, action, resourceType, resourceID, metadata); err != nil {
		uc.log.Error("audit log fallido", "action", action, "error", err)
	}
}

// List trae los eventos (solo de audit.events) y los enriquece con email/nombre de usuario en un
// solo lote (GetByIDs), no con un JOIN cross-schema ni una consulta por evento.
func (uc *UseCase) List(ctx context.Context, companyID uuid.UUID, filter domain.ListFilter) ([]*domain.Event, error) {
	events, err := uc.repo.List(ctx, companyID, filter)
	if err != nil {
		return nil, err
	}
	if uc.users == nil || len(events) == 0 {
		return events, nil
	}

	seen := make(map[uuid.UUID]bool)
	var ids []uuid.UUID
	for _, e := range events {
		if e.UserID != nil && !seen[*e.UserID] {
			seen[*e.UserID] = true
			ids = append(ids, *e.UserID)
		}
	}
	if len(ids) == 0 {
		return events, nil
	}

	users, err := uc.users.GetByIDs(ctx, ids)
	if err != nil {
		// No bloquear el listado de auditoría por un fallo al resolver nombres -- se muestran
		// los eventos igual, solo sin email/nombre.
		uc.log.Error("resolver usuarios de auditoría fallido", "error", err)
		return events, nil
	}
	byID := make(map[uuid.UUID]UserInfo, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	for _, e := range events {
		if e.UserID == nil {
			continue
		}
		if u, ok := byID[*e.UserID]; ok {
			e.UserEmail = u.Email
			e.UserName = u.Name
		}
	}
	return events, nil
}
