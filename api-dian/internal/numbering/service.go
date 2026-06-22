package numbering

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// Service centraliza la lógica de negocio de rangos de numeración.
type Service struct {
	repo Repository
}

// New crea el servicio de numeración.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// RegisterRange valida y persiste un nuevo rango de numeración.
func (s *Service) RegisterRange(ctx context.Context, nr NumberingRange) (*NumberingRange, error) {
	if err := validateRange(nr); err != nil {
		return nil, err
	}
	nr.IsActive = true
	return s.repo.Create(ctx, nr)
}

// GetRange devuelve un rango de numeración por ID.
func (s *Service) GetRange(ctx context.Context, id uuid.UUID) (*NumberingRange, error) {
	return s.repo.GetByID(ctx, id)
}

// ClaimNext reclama el siguiente consecutivo del rango, de forma atómica y segura bajo
// concurrencia. Es el único método que de verdad "gasta" un número — nunca se reintenta ni
// se reutiliza uno fallido, eso violaría el invariante de no-repetición de la DIAN.
func (s *Service) ClaimNext(ctx context.Context, id uuid.UUID) (int64, error) {
	return s.repo.ClaimNext(ctx, id)
}

// ListRanges devuelve los rangos de numeración de un emisor, opcionalmente filtrados por tipo
// de documento DIAN ("" = todos).
func (s *Service) ListRanges(ctx context.Context, issuerID uuid.UUID, dianDocumentTypeCode string) ([]*NumberingRange, error) {
	return s.repo.ListByIssuer(ctx, issuerID, dianDocumentTypeCode)
}

func validateRange(nr NumberingRange) error {
	if nr.IssuerID == uuid.Nil {
		return ErrMissingIssuer
	}
	if strings.TrimSpace(nr.DianDocumentTypeCode) == "" {
		return ErrMissingDocumentType
	}
	if strings.TrimSpace(nr.Prefix) == "" {
		return ErrEmptyPrefix
	}
	if nr.RangeFrom <= 0 || (nr.RangeTo != nil && nr.RangeFrom > *nr.RangeTo) {
		return ErrInvalidRange
	}
	switch nr.Environment {
	case EnvironmentProduccion, EnvironmentHabilitacion:
	default:
		return ErrInvalidEnvironment
	}
	return nil
}
