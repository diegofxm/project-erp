package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Supplier es un tercero proveedor de bienes o servicios.
// Pertenece a un tenant (company_id).
type Supplier struct {
	ID        uuid.UUID
	CompanyID uuid.UUID

	// Identificación fiscal
	IdentificationTypeCode string
	IdentificationNumber   string
	CheckDigit             string

	Name string

	// Clasificación tributaria DIAN
	TaxSchemeCode  string
	TaxSchemeName  string
	TaxRegimeCode  *string
	LiabilityCodes []string

	// Ubicación
	DepartmentCode   string
	MunicipalityCode string
	AddressLine      string

	// Contacto
	Email string
	Phone string

	// Condiciones comerciales
	PaymentTermsDays int // días de plazo de pago (0 = contado)

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrSupplierNotFound  = errors.New("proveedor no encontrado")
	ErrDuplicateSupplier = errors.New("ya existe un proveedor con ese número de identificación")
)
