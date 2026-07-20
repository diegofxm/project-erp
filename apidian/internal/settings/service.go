package settings

import (
	"context"
	"fmt"
	"regexp"

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

func (s *Service) UpdateBrandColor(ctx context.Context, issuerID uuid.UUID, color string) (*IssuerSettings, error) {
	if !hexColor.MatchString(color) {
		return nil, fmt.Errorf("settings: brand_color debe ser un color hex válido (#RRGGBB), recibido: %q", color)
	}
	current, err := s.repo.Get(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	current.BrandColor = color
	return s.repo.Save(ctx, *current)
}

func (s *Service) UpdatePricePerDocument(ctx context.Context, issuerID uuid.UUID, price int) (*IssuerSettings, error) {
	if price < 0 {
		return nil, fmt.Errorf("settings: price_per_document_cop no puede ser negativo")
	}
	current, err := s.repo.Get(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	current.PricePerDocumentCOP = price
	return s.repo.Save(ctx, *current)
}

// Update permite cambiar brand_color y price_per_document_cop en una sola llamada.
func (s *Service) Update(ctx context.Context, issuerID uuid.UUID, brandColor string, pricePerDoc int) (*IssuerSettings, error) {
	if brandColor != "" && !hexColor.MatchString(brandColor) {
		return nil, fmt.Errorf("settings: brand_color debe ser un color hex válido (#RRGGBB), recibido: %q", brandColor)
	}
	if pricePerDoc < 0 {
		return nil, fmt.Errorf("settings: price_per_document_cop no puede ser negativo")
	}
	current, err := s.repo.Get(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	if brandColor != "" {
		current.BrandColor = brandColor
	}
	current.PricePerDocumentCOP = pricePerDoc
	return s.repo.Save(ctx, *current)
}
