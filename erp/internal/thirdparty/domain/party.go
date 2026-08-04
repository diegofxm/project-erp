package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Role identifica en qué catálogo aparece un tercero. Un mismo Party puede tener ambos roles a
// la vez — una empresa puede ser cliente y proveedor de la misma compañía simultáneamente.
type Role string

const (
	RoleCustomer Role = "customer"
	RoleSupplier Role = "supplier"
)

// Party es un tercero (cliente y/o proveedor). Reemplaza los antiguos módulos customer/ y
// supplier/, que eran dos structs casi idénticos (24 de 26 campos iguales) mantenidos por
// separado sin ninguna razón funcional — una misma identificación no podía existir a la vez como
// cliente y proveedor sin duplicar todos sus datos fiscales. IsCustomer/IsSupplier son
// independientes, no mutuamente excluyentes.
type Party struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`

	// Identificación fiscal DIAN
	IdentificationTypeCode     string `json:"identification_type_code"`
	IdentificationNumber       string `json:"identification_number"`
	CheckDigit                 string `json:"check_digit"`
	EntityTypeCode             string `json:"entity_type_code"`
	MerchantRegistrationNumber string `json:"merchant_registration_number"`

	Name string `json:"name"`

	// Clasificación tributaria DIAN
	TaxSchemeCode  string   `json:"tax_scheme_code"`
	TaxSchemeName  string   `json:"tax_scheme_name"`
	TaxRegimeCode  *string  `json:"tax_regime_code"`
	LiabilityCodes []string `json:"liability_codes"`

	// Ubicación
	DepartmentCode     string `json:"department_code"`
	MunicipalityCode   string `json:"municipality_code"`
	AddressLine        string `json:"address_line"`
	AddressCityName    string `json:"address_city_name"`
	AddressStateName   string `json:"address_state_name"`
	AddressCountryCode string `json:"address_country_code"`
	AddressCountryName string `json:"address_country_name"`

	// Contacto
	Email string `json:"email"`
	Phone string `json:"phone"`

	// Roles — independientes entre sí.
	IsCustomer bool `json:"is_customer"`
	IsSupplier bool `json:"is_supplier"`

	// Campos específicos de rol — conviven en la misma fila porque un tercero puede ser
	// cliente y proveedor a la vez.
	CreditLimit      *float64 `json:"credit_limit"`       // solo aplica si IsCustomer; nil = sin límite
	PaymentTermsDays int      `json:"payment_terms_days"` // solo aplica si IsSupplier

	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrPartyNotFound     = errors.New("tercero no encontrado")
	ErrCustomerNotFound  = errors.New("cliente no encontrado")
	ErrSupplierNotFound  = errors.New("proveedor no encontrado")
	ErrDuplicateCustomer = errors.New("ya existe un cliente con ese número de identificación")
	ErrDuplicateSupplier = errors.New("ya existe un proveedor con ese número de identificación")
)
