package documents

import (
	"context"

	"github.com/diegofxm/apidian/internal/customers"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
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

	// ReleaseIfCurrent — ver numbering.Service.ReleaseIfCurrent. Se llama cuando un número
	// recién reclamado termina en rejected/send_error, o cuando confirmar falla ANTES de
	// llegar a intentar el envío (ej. construir o firmar el XML) — en ambos casos el número
	// nunca quedó realmente ante la DIAN, así que el siguiente intento lo puede reclamar de
	// nuevo en vez de avanzar y dejar un hueco (ver sección 9.33).
	ReleaseIfCurrent(ctx context.Context, id uuid.UUID, number int64) error
}

// CustomerPort define lo que documents necesita del catálogo de clientes — solo para
// verificar, cuando el llamador manda un CustomerID opcional, que ese cliente pertenece al
// mismo emisor (mismo criterio que NumberingPort con el rango: nunca se confía en que el
// cliente referenciado sea del emisor correcto sin comprobarlo). El catálogo en sí no
// participa en construir el XML — eso sigue siendo pass-through puro (ver model.go).
type CustomerPort interface {
	GetCustomer(ctx context.Context, id uuid.UUID) (*customers.Customer, error)
}

// CatalogPort valida payment_means/liability_codes contra catálogos DIAN que viven en JSONB
// (payment_terms/payment_methods) o en un TEXT[] sin FK posible (liability_codes) — ver
// auditoría de catálogos huérfanos, docs/apidian-architecture.md. Sin esto, un código
// inválido ahí solo se detectaba al confirmar, con la DIAN rechazando el documento — con esto
// se detecta antes, al crear/editar el borrador.
//
// unit_measures (lines[].UnitCode) NO se valida aquí a propósito: ese catálogo sigue
// incompleto (11 códigos de muestra frente al estándar UN/ECE Rec. 20 completo, ver
// migrations/000006_products.up.sql) — validar contra un catálogo incompleto rechazaría
// códigos legítimos que la DIAN sí aceptaría. Se agrega cuando se complete ese catálogo.
type CatalogPort interface {
	IsValidPaymentTerm(ctx context.Context, code string) (bool, error)
	IsValidPaymentMethod(ctx context.Context, code string) (bool, error)
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)
}
