package products

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// Service centraliza la lógica de negocio del catálogo de productos — CRUD simple. La capa
// HTTP acota cada operación al emisor autenticado (ver internal/api/handler_products.go).
type Service struct {
	repo Repository
}

// New crea el servicio de productos.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateProduct valida y persiste un producto nuevo.
func (s *Service) CreateProduct(ctx context.Context, issuerID uuid.UUID, p Product) (*Product, error) {
	if err := validateProduct(p); err != nil {
		return nil, err
	}
	p.IssuerID = issuerID
	return s.repo.Create(ctx, p)
}

// GetProduct devuelve un producto por ID.
func (s *Service) GetProduct(ctx context.Context, id uuid.UUID) (*Product, error) {
	return s.repo.GetByID(ctx, id)
}

// ListProducts devuelve los productos de un emisor.
func (s *Service) ListProducts(ctx context.Context, issuerID uuid.UUID) ([]*Product, error) {
	return s.repo.ListByIssuer(ctx, issuerID)
}

// UpdateProduct reemplaza los datos de un producto existente del emisor dado.
func (s *Service) UpdateProduct(ctx context.Context, issuerID, id uuid.UUID, p Product) (*Product, error) {
	if err := validateProduct(p); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, issuerID, id, p)
}

// DeleteProduct elimina un producto del emisor dado. No afecta documentos ya emitidos — esos
// conservan su propio snapshot de líneas (ver internal/documents).
func (s *Service) DeleteProduct(ctx context.Context, issuerID, id uuid.UUID) error {
	return s.repo.Delete(ctx, issuerID, id)
}

func validateProduct(p Product) error {
	if strings.TrimSpace(p.Description) == "" {
		return ErrEmptyDescription
	}
	if strings.TrimSpace(p.UnitCode) == "" {
		return ErrEmptyUnitCode
	}
	if p.UnitPriceCents < 0 {
		return ErrInvalidUnitPrice
	}
	return nil
}
