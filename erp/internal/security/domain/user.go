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
}
