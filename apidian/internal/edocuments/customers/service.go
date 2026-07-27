package customers

import (
	"context"
	"fmt"
	"strings"

	"github.com/diegofxm/apidian/internal/catalogs/nit"
	"github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
)

// Service centraliza la lógica de negocio del catálogo de clientes — CRUD simple, sin reglas
// más allá de validar los campos mínimos. La capa HTTP es quien acota cada operación al
// emisor autenticado (ver internal/api/handler_customers.go).
type Service struct {
	repo     Repository
	catalogs CatalogPort
}

// New crea el servicio de clientes.
func New(repo Repository, catalogPort CatalogPort) *Service {
	return &Service{repo: repo, catalogs: catalogPort}
}

// CreateCustomer valida y persiste un cliente nuevo.
func (s *Service) CreateCustomer(ctx context.Context, issuerID uuid.UUID, party domain.Party) (*Customer, error) {
	if err := s.resolveTaxSchemeName(ctx, &party); err != nil {
		return nil, err
	}
	if err := s.validateParty(ctx, party); err != nil {
		return nil, err
	}
	if err := deriveVerificationCode(&party); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, Customer{IssuerID: issuerID, Party: party})
}

// GetCustomer devuelve un cliente por ID.
func (s *Service) GetCustomer(ctx context.Context, id uuid.UUID) (*Customer, error) {
	return s.repo.GetByID(ctx, id)
}

// ListCustomers devuelve los clientes de un emisor.
func (s *Service) ListCustomers(ctx context.Context, issuerID uuid.UUID) ([]*Customer, error) {
	return s.repo.ListByIssuer(ctx, issuerID)
}

// UpdateCustomer reemplaza los datos de un cliente existente del emisor dado.
func (s *Service) UpdateCustomer(ctx context.Context, issuerID, id uuid.UUID, party domain.Party) (*Customer, error) {
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

// DeleteCustomer elimina un cliente del emisor dado. No afecta documentos ya emitidos — esos
// conservan su propio snapshot (ver model.go).
func (s *Service) DeleteCustomer(ctx context.Context, issuerID, id uuid.UUID) error {
	return s.repo.Delete(ctx, issuerID, id)
}

// deriveVerificationCode calcula el dígito de verificación módulo 11 cuando la identificación
// es un NIT ("31" — ver domain.Identification.VerificationCode, cuyo propio comentario dice
// que solo aplica a ese tipo) — el cliente ya no puede mandar un dígito que no corresponda, se
// deriva igual que TaxSchemeName (el backend deriva, no confía). Para cualquier otro tipo de
// identificación no se toca nada: el concepto de dígito de verificación no existe ahí.
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

// resolveTaxSchemeName deriva TaxSchemeName del catálogo tax_types a partir de
// TaxSchemeCode — el cliente ya no puede mandar el nombre (ver partyDTO en
// internal/api/dto.go), así ningún consumidor de la API puede guardar un código y un nombre
// que no correspondan entre sí. A diferencia de issuers (donde siempre hay un default "ZZ"),
// en customers TaxSchemeCode es opcional: vacío significa "sin régimen tributario registrado
// para este cliente", se omite la derivación.
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

// validateParty es un método de Service (no función libre) porque valida LiabilityCodes
// contra el catálogo en Postgres — liability_codes es TEXT[], sin FK posible contra cada
// elemento (ver CatalogPort en ports.go).
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
