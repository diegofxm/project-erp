package vendors

import (
	"context"
	"fmt"
	"strings"

	"github.com/diegofxm/catalogs/nit"
	"github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
)

// Service centraliza la lógica de negocio del catálogo de proveedores — mismo patrón que
// customers.Service (ver internal/customers/service.go).
type Service struct {
	repo     Repository
	catalogs CatalogPort
}

// New crea el servicio de proveedores.
func New(repo Repository, catalogPort CatalogPort) *Service {
	return &Service{repo: repo, catalogs: catalogPort}
}

// CreateVendor valida y persiste un proveedor nuevo.
func (s *Service) CreateVendor(ctx context.Context, issuerID uuid.UUID, party domain.Party) (*Vendor, error) {
	if err := s.resolveTaxSchemeName(ctx, &party); err != nil {
		return nil, err
	}
	if err := s.validateParty(ctx, party); err != nil {
		return nil, err
	}
	if err := deriveVerificationCode(&party); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, Vendor{IssuerID: issuerID, Party: party})
}

// GetVendor devuelve un proveedor por ID.
func (s *Service) GetVendor(ctx context.Context, id uuid.UUID) (*Vendor, error) {
	return s.repo.GetByID(ctx, id)
}

// ListVendors devuelve los proveedores de un emisor.
func (s *Service) ListVendors(ctx context.Context, issuerID uuid.UUID) ([]*Vendor, error) {
	return s.repo.ListByIssuer(ctx, issuerID)
}

// UpdateVendor reemplaza los datos de un proveedor existente del emisor dado.
func (s *Service) UpdateVendor(ctx context.Context, issuerID, id uuid.UUID, party domain.Party) (*Vendor, error) {
	if err := s.resolveTaxSchemeName(ctx, &party); err != nil {
		return nil, err
	}
	if err := s.validateParty(ctx, party); err != nil {
		return nil, err
	}
	if err := deriveVerificationCode(&party); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, issuerID, id, party)
}

// DeleteVendor elimina un proveedor del emisor dado.
func (s *Service) DeleteVendor(ctx context.Context, issuerID, id uuid.UUID) error {
	return s.repo.Delete(ctx, issuerID, id)
}

func deriveVerificationCode(p *domain.Party) error {
	if p.Identification.TypeCode != "31" {
		return nil
	}
	digit, err := nit.ComputeCheckDigit(p.Identification.Number)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidIdentificationNumber, err)
	}
	p.Identification.VerificationCode = digit
	return nil
}

func (s *Service) resolveTaxSchemeName(ctx context.Context, p *domain.Party) error {
	if p.TaxSchemeCode == "" {
		p.TaxSchemeName = ""
		return nil
	}
	name, found, err := s.catalogs.GetTaxTypeName(ctx, p.TaxSchemeCode)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: %q", ErrInvalidTaxSchemeCode, p.TaxSchemeCode)
	}
	p.TaxSchemeName = name
	return nil
}

func (s *Service) validateParty(ctx context.Context, p domain.Party) error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrEmptyName
	}
	if strings.TrimSpace(p.Identification.Number) == "" {
		return ErrEmptyIdentification
	}
	for _, code := range p.LiabilityCodes {
		ok, err := s.catalogs.IsValidLiabilityCode(ctx, code)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %q", ErrInvalidLiabilityCode, code)
		}
	}
	return nil
}
