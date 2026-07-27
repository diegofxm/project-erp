package customers

import (
	"context"

	"github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia del catálogo de clientes.
//
// Update/Delete reciben issuerID además del id y acotan la consulta en el propio SQL
// (WHERE id = $1 AND issuer_id = $2) — a diferencia de GetByID (que no conoce tenants, igual
// patrón que numbering.GetRange/documents.GetDocument, donde la capa HTTP compara después),
// estas son las primeras operaciones de mutación por ID de todo el proyecto: para no depender
// de que cada handler futuro recuerde comparar el dueño antes de mutar, el chequeo va en la
// query misma.
type Repository interface {
	Create(ctx context.Context, c Customer) (*Customer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Customer, error)
	ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]*Customer, error)
	Update(ctx context.Context, issuerID, id uuid.UUID, party domain.Party) (*Customer, error)
	Delete(ctx context.Context, issuerID, id uuid.UUID) error
}
