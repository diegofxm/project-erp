package auth

import (
	"time"

	"github.com/google/uuid"
)

// RoleAdmin es el único rol de USUARIO que existe hoy — no hay roles granulares de
// solo-lectura, etc. todavía. Agregar un rol nuevo más adelante es agregar un valor aquí y un
// chequeo donde haga falta, no una migración de esquema.
const RoleAdmin = "admin"

// RoleOwner es el rol que se asigna en user_issuers cuando un usuario CREA una empresa (ver
// CreateIssuerForUser) — distinto de RoleAdmin (que es del usuario en general, no de su
// relación con una empresa puntual). No se hace cumplir todavía (ver migración 000009):
// cualquier vínculo da acceso completo.
const RoleOwner = "owner"

// User es la cuenta de acceso. Ya NO tiene un IssuerID fijo — un usuario puede no tener
// todavía ninguna empresa vinculada (registro desacoplado) o tener varias (multi-empresa/
// sucursales), ver user_issuers y docs/apidian-architecture.md sección 9.32. Cuál empresa
// está "activa" en un momento dado es contextual (vive en el token, no en este struct) — ver
// TokenIssuer.Issue.
//
// PasswordHash vacío significa que el usuario fue creado por invitación y aún no ha
// configurado su contraseña — no puede iniciar sesión hasta aceptar la invitación.
type User struct {
	ID                   uuid.UUID
	Email                string
	PasswordHash         string // vacío = sin contraseña (usuario invitado)
	Name                 string
	Role                 string
	IsSuperAdmin         bool
	IsActive             bool
	InviteToken          *uuid.UUID
	InviteTokenExpiresAt *time.Time
	InviteAcceptedAt     *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// IssuerMembership es una fila de user_issuers — el vínculo entre un usuario y una empresa
// a la que tiene acceso. Role no se hace cumplir todavía (ver migración 000009): cualquier
// vínculo da acceso completo, igual que el modelo anterior de "un usuario = un emisor".
type IssuerMembership struct {
	UserID    uuid.UUID
	IssuerID  uuid.UUID
	Role      string
	CreatedAt time.Time
}
