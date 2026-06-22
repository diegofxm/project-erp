package auth

import (
	"context"

	"github.com/diegofxm/api-dian/internal/issuers"
)

// IssuerPort define lo que auth necesita del dominio de emisores: crear el emisor como parte
// del registro de su primer usuario admin, en una sola llamada (POST /api/v1/auth/register).
// Interfaz angosta para poder probar Service con un fake, mismo patrón que
// documents.IssuerPort.
type IssuerPort interface {
	RegisterIssuer(ctx context.Context, iss issuers.Issuer) (*issuers.Issuer, error)
}
