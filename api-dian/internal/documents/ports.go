package documents

import (
	"context"

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
