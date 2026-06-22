package auth

import (
	"time"

	"github.com/google/uuid"
)

// RoleAdmin es el único rol que existe hoy — todo usuario administra por completo su propio
// emisor (no hay roles granulares de solo-lectura, etc. todavía). Agregar un rol nuevo más
// adelante es agregar un valor aquí y un chequeo donde haga falta, no una migración de esquema.
const RoleAdmin = "admin"

// User es la cuenta de acceso de un emisor/tenant DIAN. "Un usuario = un emisor" — IssuerID es
// obligatorio y fijo: no existen usuarios sin emisor ni usuarios con varios emisores (decisión
// explícita del usuario, ver docs/api-dian-architecture.md sección 9.17). Si en el futuro hace
// falta que una misma persona administre varios emisores (ej. un contador con varios clientes),
// el camino natural es una tabla intermedia user_issuers, no forzarlo en este struct ahora.
type User struct {
	ID           uuid.UUID
	IssuerID     uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	Role         string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
