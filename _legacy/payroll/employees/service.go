package employees

import (
	"context"

	"github.com/google/uuid"
)

// Service encapsula la lógica de gestión de empleados.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*Employee, error) {
	return s.repo.Create(ctx, in)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Employee, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, companyID uuid.UUID) ([]*Employee, error) {
	return s.repo.List(ctx, companyID)
}

func (s *Service) Deactivate(ctx context.Context, id uuid.UUID) error {
	return s.repo.Deactivate(ctx, id)
}
