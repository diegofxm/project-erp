package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type User struct {
	ID                   uuid.UUID
	Email                string
	PasswordHash         *string // nil = invitación pendiente
	Name                 string
	Role                 Role
	IsSuperAdmin         bool
	IsActive             bool
	InviteToken          *uuid.UUID
	InviteTokenExpiresAt *time.Time
	InviteAcceptedAt     *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// UserCompany vincula un usuario a una empresa con su rol en ella.
type UserCompany struct {
	UserID    uuid.UUID
	CompanyID uuid.UUID
	Role      Role
	CreatedAt time.Time
}

// AuthResult es lo que devuelven los casos de uso de autenticación.
type AuthResult struct {
	User      User
	Token     string
	CompanyID uuid.UUID // uuid.Nil si el usuario aún no tiene empresa activa
	// Role es el rol RESUELTO por empresa (owner/admin/member, ver user_companies.role) — el mismo
	// valor que ya se incrusta en el JWT al emitirlo. Deliberadamente separado de User.Role: ese
	// campo del usuario es solo un default de alta (siempre "admin" al registrarse o ser invitado,
	// ver RegisterUseCase/InviteUserUseCase) y NUNCA representa el rol real dentro de una empresa
	// — usarlo para eso fue el bug que dejaba a cualquier member viendo botones de gestión.
	// Vacío si el usuario aún no tiene empresa activa (igual que CompanyID = uuid.Nil).
	Role string
}
