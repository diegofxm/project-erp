package subscriptions

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrLimitReached = errors.New("subscription: límite mensual de documentos alcanzado")

type Service struct {
	repo *PostgresRepository
}

func NewService(repo *PostgresRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetActive(ctx context.Context, issuerID uuid.UUID) (*Subscription, error) {
	return s.repo.GetActive(ctx, issuerID)
}

func (s *Service) ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]Subscription, error) {
	return s.repo.ListByIssuer(ctx, issuerID)
}

func (s *Service) Assign(ctx context.Context, issuerID, planID uuid.UUID) (*Subscription, error) {
	return s.repo.Assign(ctx, issuerID, planID)
}

// CheckLimit verifica si el emisor puede crear un documento más este mes.
// Si no tiene suscripción activa se permite (graceful: sin plan = sin límite).
// Si el plan es ilimitado (MaxDocumentsPerMonth nil) se permite siempre.
func (s *Service) CheckLimit(ctx context.Context, issuerID uuid.UUID) error {
	sub, err := s.repo.GetActive(ctx, issuerID)
	if errors.Is(err, ErrNotFound) {
		return nil // sin suscripción = sin límite
	}
	if err != nil {
		return fmt.Errorf("subscriptions: check limit: %w", err)
	}
	if sub.MaxDocumentsPerMonth == nil {
		return nil // plan ilimitado
	}

	count, err := s.repo.CountDocumentsThisMonth(ctx, issuerID)
	if err != nil {
		return fmt.Errorf("subscriptions: count documents: %w", err)
	}
	if count >= *sub.MaxDocumentsPerMonth {
		return fmt.Errorf("%w: %d/%d documentos este mes (plan %s)",
			ErrLimitReached, count, *sub.MaxDocumentsPerMonth, sub.PlanName)
	}
	return nil
}
