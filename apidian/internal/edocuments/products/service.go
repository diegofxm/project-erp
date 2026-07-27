package products

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Service centraliza la lógica de negocio del catálogo de productos — CRUD simple. La capa
// HTTP acota cada operación al emisor autenticado (ver internal/api/handler_products.go).
type Service struct {
	repo     Repository
	catalogs CatalogPort
}

// New crea el servicio de productos.
func New(repo Repository, catalogPort CatalogPort) *Service {
	return &Service{repo: repo, catalogs: catalogPort}
}

// CreateProduct valida y persiste un producto nuevo.
func (s *Service) CreateProduct(ctx context.Context, issuerID uuid.UUID, p Product) (*Product, error) {
	if err := s.resolveTaxTypeName(ctx, &p); err != nil {
		return nil, err
	}
	if err := s.resolveItemStandard(ctx, &p); err != nil {
		return nil, err
	}
	if err := validateProduct(p); err != nil {
		return nil, err
	}
	p.IssuerID = issuerID
	return s.repo.Create(ctx, p)
}

// resolveTaxTypeName deriva TaxTypeName del catálogo tax_types a partir de TaxTypeCode — el
// cliente ya no puede mandar el nombre (ver productRequest en
// internal/api/handler_products.go). TaxTypeCode es opcional (un producto puede no tener
// impuesto por defecto): vacío omite la derivación.
func (s *Service) resolveTaxTypeName(ctx context.Context, p *Product) error {
	if p.TaxTypeCode == "" {
		p.TaxTypeName = ""
		return nil
	}
	name, found, err := s.catalogs.GetTaxTypeName(ctx, p.TaxTypeCode)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: %q", ErrInvalidTaxTypeCode, p.TaxTypeCode)
	}
	p.TaxTypeName = name
	return nil
}

// resolveItemStandard deriva ItemTypeName/ItemTypeAgencyID del catálogo item_standards a
// partir de ItemTypeCode (tabla 13.3.5 del Anexo Técnico, ver
// docs/apidian-architecture.md sección 9.45) — el cliente ya no puede mandarlos (ver
// productRequest en internal/api/handler_products.go). A diferencia de TaxTypeCode (opcional,
// "sin impuesto por defecto" es válido), ItemTypeCode vacío defaultea a "999" (estándar propio
// del contribuyente) — la DIAN exige que cac:StandardItemIdentification siempre tenga una
// clasificación válida, nunca "ninguna".
func (s *Service) resolveItemStandard(ctx context.Context, p *Product) error {
	if p.ItemTypeCode == "" {
		p.ItemTypeCode = "999"
	}
	name, found, err := s.catalogs.GetItemStandardName(ctx, p.ItemTypeCode)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: %q", ErrInvalidItemStandardCode, p.ItemTypeCode)
	}
	agencyID, _, err := s.catalogs.GetItemStandardAgencyID(ctx, p.ItemTypeCode)
	if err != nil {
		return err
	}
	p.ItemTypeName = name
	p.ItemTypeAgencyID = agencyID
	return nil
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
	if err := s.resolveTaxTypeName(ctx, &p); err != nil {
		return nil, err
	}
	if err := s.resolveItemStandard(ctx, &p); err != nil {
		return nil, err
	}
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
