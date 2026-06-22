package customers

import (
	"context"
	"strings"

	"github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
)

// Service centraliza la lógica de negocio del catálogo de clientes — CRUD simple, sin reglas
// más allá de validar los campos mínimos. La capa HTTP es quien acota cada operación al
// emisor autenticado (ver internal/api/handler_customers.go).
type Service struct {
	repo Repository
}

// New crea el servicio de clientes.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateCustomer valida y persiste un cliente nuevo.
func (s *Service) CreateCustomer(ctx context.Context, issuerID uuid.UUID, party domain.Party) (*Customer, error) {
	if err := validateParty(party); err != nil {
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
	if err := validateParty(party); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, issuerID, id, party)
}

// DeleteCustomer elimina un cliente del emisor dado. No afecta documentos ya emitidos — esos
// conservan su propio snapshot (ver model.go).
func (s *Service) DeleteCustomer(ctx context.Context, issuerID, id uuid.UUID) error {
	return s.repo.Delete(ctx, issuerID, id)
}

func validateParty(p domain.Party) error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrEmptyName
	}
	if strings.TrimSpace(p.Identification.Number) == "" {
		return ErrEmptyIdentification
	}
	return nil
}
