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
//
// nextNumber es opcional — nil significa "rango nuevo, nunca usado": el primer ClaimNext
// entrega RangeFrom, igual que siempre. Si la resolución YA tiene números autorizados de
// verdad en la DIAN (ej. se usó antes directamente con cofacture, o ya se emitió manualmente
// fuera de apidian), nextNumber le dice exactamente qué número debe entregar el primer
// ClaimNext, para no reclamar uno que la DIAN ya vio y autorizó — un duplicado real, no solo
// un error de apidian.
func (s *Service) RegisterRange(ctx context.Context, nr NumberingRange, nextNumber *int64) (*NumberingRange, error) {
	if err := validateRange(nr); err != nil {
		return nil, err
	}
	if nextNumber != nil {
		if *nextNumber < nr.RangeFrom || (nr.RangeTo != nil && *nextNumber > *nr.RangeTo) {
			return nil, ErrNextNumberOutOfRange
		}
		nr.CurrentNumber = *nextNumber - 1
	} else {
		nr.CurrentNumber = nr.RangeFrom - 1
	}
	nr.IsActive = true
	return s.repo.Create(ctx, nr)
}

// GetRange devuelve un rango de numeración por ID.
func (s *Service) GetRange(ctx context.Context, id uuid.UUID) (*NumberingRange, error) {
	return s.repo.GetByID(ctx, id)
}

// ClaimNext reclama el siguiente consecutivo del rango, de forma atómica y segura bajo
// concurrencia. ClaimNext en sí mismo nunca reintenta ni reutiliza nada — siempre avanza. Ver
// ReleaseIfCurrent para el único mecanismo que sí puede hacer que un número se vuelva a
// entregar (porque nunca quedó realmente gastado ante la DIAN, no porque se esté repitiendo
// uno ya válido).
func (s *Service) ClaimNext(ctx context.Context, id uuid.UUID) (int64, error) {
	return s.repo.ClaimNext(ctx, id)
}

// ReleaseIfCurrent revierte el último ClaimNext de id — pero solo si number sigue siendo el
// número vigente del rango (nadie reclamó otro desde entonces). Pensado para un documento
// rechazado por la DIAN o que nunca se logró transmitir (send_error): ahí el número nunca
// quedó realmente registrado, así que el siguiente intento lo puede volver a reclamar en vez
// de avanzar y dejar un hueco permanente (ver docs/apidian-architecture.md sección 9.33).
func (s *Service) ReleaseIfCurrent(ctx context.Context, id uuid.UUID, number int64) error {
	return s.repo.ReleaseIfCurrent(ctx, id, number)
}

// ClearTestSetID — ver Repository.ClearTestSetID.
func (s *Service) ClearTestSetID(ctx context.Context, id uuid.UUID) error {
	return s.repo.ClearTestSetID(ctx, id)
}

// DeactivateRange — ver Repository.Deactivate.
func (s *Service) DeactivateRange(ctx context.Context, id uuid.UUID) error {
	return s.repo.Deactivate(ctx, id)
}

// ActivateRange — ver Repository.Activate.
func (s *Service) ActivateRange(ctx context.Context, id uuid.UUID) error {
	return s.repo.Activate(ctx, id)
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
