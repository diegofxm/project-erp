package plans

import (
	"context"
	"math"

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

// ApplyIncrement aplica el incremento anual configurado al plan (affiliation_fee, renewal_fee,
// price_cop) redondeando al peso más cercano. Si annual_increment_pct es 0, devuelve el plan
// sin cambios.
func (s *Service) ApplyIncrement(ctx context.Context, id uuid.UUID) (*Plan, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.AnnualIncrementPct <= 0 {
		return p, nil
	}
	factor := 1.0 + p.AnnualIncrementPct/100.0
	if p.AffiliationFeeCOP > 0 {
		p.AffiliationFeeCOP = int(math.Round(float64(p.AffiliationFeeCOP) * factor))
	}
	if p.RenewalFeeCOP > 0 {
		p.RenewalFeeCOP = int(math.Round(float64(p.RenewalFeeCOP) * factor))
	}
	if p.PriceCOP > 0 {
		p.PriceCOP = int(math.Round(float64(p.PriceCOP) * factor))
	}
	return s.repo.Update(ctx, *p)
}
