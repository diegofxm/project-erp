package issuers

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia del dominio de emisores.
type Repository interface {
	Create(ctx context.Context, iss Issuer) (*Issuer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Issuer, error)
	GetByNIT(ctx context.Context, nit string) (*Issuer, error)

	// Update persiste software_id/software_pin/certificate/certificate_password/logo — los
	// únicos campos que UpdateIssuer modifica (ver service.go). El resto del emisor (NIT, razón
	// social, ubicación, etc.) no se actualiza por este camino.
	Update(ctx context.Context, iss Issuer) (*Issuer, error)

	// UpdateProfile persiste los campos de perfil/identidad editables (razón social, nombre
	// comercial, dirección, correo, teléfono, códigos fiscales) — todo lo que NO es secreto
	// cifrado ni inmutable (NIT/ambiente/tipo_identificación). Los secretos siguen
	// exclusivamente en Update.
	UpdateProfile(ctx context.Context, iss Issuer) (*Issuer, error)
}
