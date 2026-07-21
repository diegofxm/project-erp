package auth

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia del dominio de autenticación.
type Repository interface {
	Create(ctx context.Context, u User) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, u User) (*User, error)

	// LinkIssuer vincula a userID con issuerID (user_issuers) — el camino para que un usuario
	// gane acceso a una empresa, sea porque la crea (rol "owner") o porque se le da acceso a
	// una ya existente (gradual, multi-empresa). Idempotente: vincular dos veces el mismo par
	// no debe fallar ni duplicar la fila (ver migración 000009, PK compuesta).
	LinkIssuer(ctx context.Context, userID, issuerID uuid.UUID, role string) error

	// ListIssuerIDs devuelve los emisores a los que userID tiene acceso — para listar
	// "mis empresas" y para decidir si el login puede auto-seleccionar la activa (exactamente
	// una) o debe dejarlo sin seleccionar (cero o varias).
	ListIssuerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)

	// HasAccess confirma que userID tiene un vínculo con issuerID — se usa antes de emitir un
	// token con ese tenant activo (POST /issuers/{id}/select), nunca confiar en que el cliente
	// realmente tiene acceso al ID que mandó.
	HasAccess(ctx context.Context, userID, issuerID uuid.UUID) (bool, error)

	// GetByInviteToken busca un usuario por su token de invitación — usado para validar el link
	// del correo de bienvenida antes de que el usuario configure su contraseña.
	GetByInviteToken(ctx context.Context, token uuid.UUID) (*User, error)

	// SetPassword establece la contraseña, limpia el token de invitación y registra
	// invite_accepted_at=NOW() en una sola operación atómica.
	SetPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error

	// ListAll devuelve todos los usuarios del sistema — solo para el panel de superadmin.
	ListAll(ctx context.Context) ([]User, error)
}
