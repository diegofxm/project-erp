package periods

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Service encapsula la lógica de periodos contables.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetOrCreate devuelve el periodo para el mes/año de la fecha dada, creándolo si no existe.
// El motor de asientos lo llama en cada Post para no obligar al usuario a abrir periodos manualmente.
func (s *Service) GetOrCreate(ctx context.Context, companyID uuid.UUID, date time.Time) (*AccountingPeriod, error) {
	p, err := s.repo.GetByYearMonth(ctx, companyID, date.Year(), int(date.Month()))
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, ErrPeriodNotFound) {
		return nil, err
	}
	return s.repo.Create(ctx, AccountingPeriod{
		CompanyID: companyID,
		Year:      date.Year(),
		Month:     int(date.Month()),
	})
}

// GetByID devuelve un periodo por UUID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*AccountingPeriod, error) {
	return s.repo.GetByID(ctx, id)
}

// List devuelve los periodos de una empresa, del más reciente al más antiguo.
func (s *Service) List(ctx context.Context, companyID uuid.UUID) ([]*AccountingPeriod, error) {
	return s.repo.List(ctx, companyID)
}

// Close cierra el periodo — no se pueden registrar más asientos en él.
func (s *Service) Close(ctx context.Context, id uuid.UUID) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if p.Status == StatusClosed {
		return ErrPeriodClosed
	}
	return s.repo.Close(ctx, id)
}
