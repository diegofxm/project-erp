// Package customer gestiona el catálogo de clientes del emisor.
// Patrón completo en: domain/ application/ infrastructure/ interfaces/
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID
	CompanyID uuid.UUID // tenant

	// Identificación
	IdentificationNumber   string
	IdentificationTypeCode string
	VerificationCode       string // dígito de verificación NIT (derivado, no ingresado)

	// Datos fiscales
	Name           string
	TaxSchemeCode  string
	TaxSchemeName  string
	LiabilityCodes []string
	TaxRegimeCode  string

	// Contacto y ubicación
	Email            string
	Phone            string
	AddressLine      string
	MunicipalityCode string
	DepartmentCode   string

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Repository interface {
	Save(ctx context.Context, c Customer) (*Customer, error)
	Update(ctx context.Context, c Customer) (*Customer, error)
	Delete(ctx context.Context, companyID, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Customer, error)
	ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*Customer, error)
}

// CatalogPort es lo que Customer necesita de catalog/ — no importa catalog/ directamente.
type CatalogPort interface {
	GetTaxTypeName(code string) (string, bool, error)
	IsValidLiabilityCode(code string) (bool, error)
}
