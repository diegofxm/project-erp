package auth

import (
	"context"

	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/google/uuid"
)

// IssuerPort define lo que auth necesita del dominio de emisores: crear una empresa
// (CreateIssuerForUser) y consultarla por ID para devolverla como "empresa activa" en
// Login/CreateIssuerForUser/ListUserIssuers/SelectIssuer. Interfaz angosta para poder probar
// Service con un fake, mismo patrón que documents.IssuerPort.
type IssuerPort interface {
	RegisterIssuer(ctx context.Context, iss issuers.Issuer) (*issuers.Issuer, error)
	GetIssuer(ctx context.Context, id uuid.UUID) (*issuers.Issuer, error)
}
