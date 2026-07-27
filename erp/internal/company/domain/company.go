// Package company gestiona la entidad raíz del sistema: la empresa (tenant).
// Cada registro de cualquier otro módulo lleva un company_id que apunta aquí.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Company struct {
	ID   uuid.UUID
	NIT  string
	Name string // razón social

	// Datos fiscales
	IdentificationTypeCode string // "31" NIT
	CheckDigit             string
	TaxSchemeCode          string // "ZZ" no aplica, "01" IVA
	TaxSchemeName          string
	LiabilityCodes         []string // ["R-99-PN", ...]
	TaxRegimeCode          string

	// Ubicación
	DepartmentCode   string
	MunicipalityCode string
	AddressLine      string

	// Contacto
	Email string
	Phone string

	// Ambiente DIAN
	Environment string // "1" producción, "2" habilitación

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Repository interface {
	Save(company Company) (*Company, error)
	GetByID(id uuid.UUID) (*Company, error)
	GetByNIT(nit string) (*Company, error)
}
