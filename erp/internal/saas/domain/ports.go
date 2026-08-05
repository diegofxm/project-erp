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

// UserPort lista los usuarios de la plataforma (todas las empresas) — para GET /admin/users, la
// vista de superadmin de "quién tiene cuenta". Puerto local sobre security/domain.Repository.
type UserPort interface {
	ListAll(ctx context.Context) ([]PlatformUser, error)
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
