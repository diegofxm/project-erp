package plans

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo *PostgresRepository
}

func NewService(repo *PostgresRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]Plan, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Plan, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetFree(ctx context.Context) (*Plan, error) {
	return s.repo.GetFree(ctx)
}

func (s *Service) Create(ctx context.Context, p Plan) (*Plan, error) {
	return s.repo.Create(ctx, p)
}

func (s *Service) Update(ctx context.Context, p Plan) (*Plan, error) {
	return s.repo.Update(ctx, p)
}
