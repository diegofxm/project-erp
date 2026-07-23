package audit

import (
	"context"

	"github.com/google/uuid"
)

// Service provee logging y consulta de eventos de auditoría.
type Service struct {
	repo Repository
}

// New construye el servicio.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// Log persiste un evento de auditoría. No propaga el error al llamador — un fallo de
// auditoría nunca debe interrumpir la operación principal.
func (s *Service) Log(ctx context.Context, e Event) {
	_ = s.repo.Insert(ctx, e)
}

// List devuelve eventos del emisor, opcionalmente filtrados por resource_id.
func (s *Service) List(ctx context.Context, issuerID uuid.UUID, filter ListFilter) ([]Event, error) {
	return s.repo.List(ctx, issuerID, filter)
}
