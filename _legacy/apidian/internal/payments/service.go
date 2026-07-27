package payments

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

func (s *Service) Record(ctx context.Context, issuerID uuid.UUID, ptype PaymentType, amountCOP int, note string) (*Payment, error) {
	return s.repo.Create(ctx, Payment{
		IssuerID:  issuerID,
		Type:      ptype,
		AmountCOP: amountCOP,
		Note:      note,
	})
}

func (s *Service) ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]Payment, error) {
	return s.repo.ListByIssuer(ctx, issuerID)
}
