package documents

import (
	"context"

	"github.com/diegofxm/api-dian/internal/customers"
	"github.com/diegofxm/api-dian/internal/issuers"
	"github.com/diegofxm/api-dian/internal/numbering"
	"github.com/google/uuid"
)

// IssuerPort define lo que documents necesita del dominio de emisores. Es una interfaz
// angosta (no *issuers.Service directamente) para poder probar Service con un fake en tests,
// igual patrón que transfers.AccountPort en core-bank.
type IssuerPort interface {
	GetIssuer(ctx context.Context, id uuid.UUID) (*issuers.Issuer, error)
}

// NumberingPort define lo que documents necesita del dominio de numeración.
type NumberingPort interface {
	GetRange(ctx context.Context, id uuid.UUID) (*numbering.NumberingRange, error)
	ClaimNext(ctx context.Context, id uuid.UUID) (int64, error)
}

// CustomerPort define lo que documents necesita del catálogo de clientes — solo para
// verificar, cuando el llamador manda un CustomerID opcional, que ese cliente pertenece al
// mismo emisor (mismo criterio que NumberingPort con el rango: nunca se confía en que el
// cliente referenciado sea del emisor correcto sin comprobarlo). El catálogo en sí no
// participa en construir el XML — eso sigue siendo pass-through puro (ver model.go).
type CustomerPort interface {
	GetCustomer(ctx context.Context, id uuid.UUID) (*customers.Customer, error)
}
