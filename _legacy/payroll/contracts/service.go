package contracts

import (
	"context"

	"github.com/google/uuid"
)

// Service encapsula la lógica de gestión de contratos laborales.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*Contract, error) {
	// Verificar que el empleado no tenga ya un contrato activo.
	_, err := s.repo.GetActive(ctx, in.EmployeeID)
	if err == nil {
		return nil, ErrAlreadyActive
	}
	if err != ErrNotFound {
		return nil, err
	}
	return s.repo.Create(ctx, in)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Contract, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) GetActive(ctx context.Context, employeeID uuid.UUID) (*Contract, error) {
	return s.repo.GetActive(ctx, employeeID)
}

func (s *Service) List(ctx context.Context, companyID uuid.UUID) ([]*Contract, error) {
	return s.repo.List(ctx, companyID)
}

func (s *Service) Terminate(ctx context.Context, id uuid.UUID, cause string) error {
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if !c.IsActive {
		return ErrNotActive
	}
	return s.repo.Terminate(ctx, id, cause)
}
