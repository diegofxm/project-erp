package settings

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var hexColor = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type Service struct {
	repo *PostgresRepository
}

func NewService(repo *PostgresRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, issuerID uuid.UUID) (*IssuerSettings, error) {
	return s.repo.Get(ctx, issuerID)
}

// GetBrandColor devuelve el color de marca del emisor, o el default si no hay configuración.
// Nunca propaga errores — en el peor caso el PDF usa el color por defecto.
func (s *Service) GetBrandColor(ctx context.Context, issuerID uuid.UUID) string {
	st, err := s.repo.Get(ctx, issuerID)
	if err != nil {
		return DefaultBrandColor
	}
	return st.BrandColor
}

// Update actualiza brand_color y las tarifas configurables sin tocar las fechas de afiliación.
func (s *Service) Update(ctx context.Context, issuerID uuid.UUID, brandColor string, pricePerDoc, affiliationFee, renewalFee int) (*IssuerSettings, error) {
	if brandColor != "" && !hexColor.MatchString(brandColor) {
		return nil, fmt.Errorf("settings: brand_color debe ser un color hex válido (#RRGGBB), recibido: %q", brandColor)
	}
	if pricePerDoc < 0 {
		return nil, fmt.Errorf("settings: price_per_document_cop no puede ser negativo")
	}
	if affiliationFee < 0 {
		return nil, fmt.Errorf("settings: affiliation_fee_cop no puede ser negativo")
	}
	if renewalFee < 0 {
		return nil, fmt.Errorf("settings: renewal_fee_cop no puede ser negativo")
	}
	current, err := s.repo.Get(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	if brandColor != "" {
		current.BrandColor = brandColor
	}
	current.PricePerDocumentCOP = pricePerDoc
	current.AffiliationFeeCOP = affiliationFee
	current.RenewalFeeCOP = renewalFee
	return s.repo.Save(ctx, *current)
}

// Affiliate registra la afiliación inicial del emisor: fija affiliated_at = ahora,
// renewal_due_at = ahora + 1 año, y guarda la tarifa de afiliación pagada.
func (s *Service) Affiliate(ctx context.Context, issuerID uuid.UUID, feePaidCOP int) (*IssuerSettings, error) {
	current, err := s.repo.Get(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	yearLater := now.AddDate(1, 0, 0)
	current.AffiliationFeeCOP = feePaidCOP
	current.AffiliatedAt = &now
	current.RenewalDueAt = &yearLater
	return s.repo.Save(ctx, *current)
}

// Renew extiende la vigencia del emisor en 1 año a partir de la fecha de vencimiento actual
// (o desde hoy si ya venció o no tiene fecha). Guarda la tarifa de renovación pagada.
func (s *Service) Renew(ctx context.Context, issuerID uuid.UUID, feePaidCOP int) (*IssuerSettings, error) {
	current, err := s.repo.Get(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	base := time.Now().UTC()
	if current.RenewalDueAt != nil && current.RenewalDueAt.After(base) {
		base = *current.RenewalDueAt
	}
	yearLater := base.AddDate(1, 0, 0)
	current.RenewalFeeCOP = feePaidCOP
	current.RenewalDueAt = &yearLater
	return s.repo.Save(ctx, *current)
}

// UpdateBrandColor es una conveniencia para actualizar solo el color de marca.
func (s *Service) UpdateBrandColor(ctx context.Context, issuerID uuid.UUID, color string) (*IssuerSettings, error) {
	current, err := s.repo.Get(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	return s.Update(ctx, issuerID, color, current.PricePerDocumentCOP, current.AffiliationFeeCOP, current.RenewalFeeCOP)
}
