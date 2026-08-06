package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// DocumentCounterPort cuenta los documentos electrónicos emitidos por una empresa en un rango de
// fechas — base para calcular cupo consumido y excedente a facturar. Puerto local (mismo patrón
// que sales/domain.CustomerPort): implementado en saas/infrastructure/electronic envolviendo
// electronic/domain.DocumentRepository, sin acoplar los dos módulos directamente.
type DocumentCounterPort interface {
	CountInPeriod(ctx context.Context, companyID uuid.UUID, from, to time.Time) (int, error)
}

// CompanyPort resuelve los datos de una empresa (razón social, NIT) para los reportes de
// facturación/renovaciones del panel superadmin — mismo patrón que sales/domain.CompanyPort. No
// expone un "listar todas las empresas": los reportes parten de
// SubscriptionRepository.ListAllActive/ListUpcomingRenewals (empresas CON suscripción) y resuelven
// el nombre/NIT de cada una puntualmente con GetCompany.
type CompanyPort interface {
	GetCompany(ctx context.Context, companyID uuid.UUID) (*CompanyInfo, error)
}

// CompanyInfo es la foto mínima de una empresa que saas necesita — evita importar
// company/domain.Company completo (que trae credenciales de certificado, etc.).
type CompanyInfo struct {
	ID           uuid.UUID
	BusinessName string
	TradeName    string
	NIT          string
}

// UserPort resuelve todo lo relacionado a cuentas de usuario para el panel de superadmin —
// listado, promoción, y aprovisionamiento de dueños nuevos. Puerto local sobre
// security/domain.Repository.
type UserPort interface {
	ListAll(ctx context.Context) ([]PlatformUser, error)
	// SetSuperAdmin promueve/degrada el flag de plataforma — PATCH /admin/users/{id}.
	SetSuperAdmin(ctx context.Context, userID uuid.UUID, value bool) error
	// InviteOwner crea (o reutiliza, si la invitación anterior no se aceptó todavía) un usuario
	// invitado sin empresa — el llamador debe crear la empresa después usando el userID devuelto
	// como dueño (ver CompanyProvisioningPort.CreateCompanyForOwner). Devuelve el token de
	// invitación en texto plano para incluirlo en el correo.
	InviteOwner(ctx context.Context, email, name string) (userID uuid.UUID, inviteToken string, err error)
}

// CompanyProvisioningPort crea la primera empresa de un dueño nuevo — usado al aprobar un
// prospecto (ver saas/application/prospect.go). Puerto local sobre company/application.CreateUseCase,
// que ya resuelve defaults fiscales + vincula al creador como "owner" + crea la bodega por defecto.
type CompanyProvisioningPort interface {
	CreateCompanyForOwner(ctx context.Context, ownerID uuid.UUID, businessName, nit string, contactEmail string) (companyID uuid.UUID, err error)
}

type PlatformUser struct {
	ID               uuid.UUID
	Email            string
	Name             string
	Role             string
	IsSuperAdmin     bool
	IsActive         bool
	InviteAcceptedAt *time.Time
	CreatedAt        time.Time
}
