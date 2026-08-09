package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository es el puerto de persistencia del módulo security.
type Repository interface {
	Save(ctx context.Context, u User) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByInviteToken(ctx context.Context, token uuid.UUID) (*User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, name, email string) (*User, error)
	// SetPassword la usa AcceptInviteUseCase — además de guardar el hash, marca la invitación como
	// aceptada (limpia el token, sella invite_accepted_at). Para un usuario ya activo que cambia su
	// propia contraseña usa UpdatePassword, que no toca nada de eso.
	SetPassword(ctx context.Context, id uuid.UUID, hash string) error
	// UpdatePassword cambia la contraseña de un usuario ya activo (ver ChangePasswordUseCase) —
	// solo actualiza el hash, sin tocar el estado de invitación.
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error
	List(ctx context.Context) ([]User, error)
	// SetSuperAdmin promueve/degrada el flag transversal de plataforma — separado de cualquier rol
	// por empresa (ver shared/tenant.IsSuperAdmin). Solo debe llamarse desde un endpoint ya
	// protegido con requireSuperAdmin (ver saas/interfaces/http) o desde el seed del primer
	// superadmin al arrancar el servidor.
	SetSuperAdmin(ctx context.Context, userID uuid.UUID, value bool) error
	// SetInviteToken regenera el token de invitación de un usuario que aún no acepta (ver
	// InviteOwnerUseCase, reintento de una aprobación de prospecto que falló a mitad de camino).
	SetInviteToken(ctx context.Context, userID uuid.UUID, token uuid.UUID, expiresAt time.Time) error
	// IncrementTokenVersion revoca todas las sesiones activas del usuario -- cualquier JWT ya
	// emitido deja de pasar la verificación en el próximo request (ver SessionVerifier). Se usa en
	// logout y al cambiar contraseña.
	IncrementTokenVersion(ctx context.Context, userID uuid.UUID) error

	// Vínculos usuario↔empresa (company_id es UUID puro; FK a company schema se añade cuando exista)
	AddCompany(ctx context.Context, userID, companyID uuid.UUID, role string) error
	ListCompanyIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	HasCompany(ctx context.Context, userID, companyID uuid.UUID) (bool, error)
	// GetRole devuelve el rol del usuario en esa empresa ("owner"/"admin"/"member") — usado para
	// incrustar el rol en el JWT al emitirlo (login/select-company), y así el RBAC del lado del
	// servidor (shared/tenant.CanManage) no necesita una consulta a BD en cada request.
	GetRole(ctx context.Context, userID, companyID uuid.UUID) (string, error)
}

// TokenService es el puerto de firma y verificación de JWT. tokenVersion (ver User.TokenVersion)
// viaja como claim propio -- Verify lo devuelve crudo, sin comparar contra nada; la comparación
// contra el valor actual en BD la hace SessionVerifier, no este servicio (que es pura
// criptografía, sin acceso a repositorio).
type TokenService interface {
	Issue(userID, companyID uuid.UUID, role string, isSuperAdmin bool, tokenVersion int) (string, error)
	Verify(raw string) (userID, companyID uuid.UUID, role string, isSuperAdmin bool, tokenVersion int, err error)
}
