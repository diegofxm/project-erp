package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Customer es un tercero receptor de documentos electrónicos (comprador, beneficiario).
// Pertenece a un tenant (company_id).
type Customer struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`

	// Identificación fiscal DIAN
	IdentificationTypeCode      string `json:"identification_type_code"`
	IdentificationNumber        string `json:"identification_number"`
	CheckDigit                  string `json:"check_digit"`
	EntityTypeCode              string `json:"entity_type_code"`
	MerchantRegistrationNumber  string `json:"merchant_registration_number"`

	// Nombre
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

	// CreditLimit — cupo máximo de cartera (ventas confirmadas sin pagar) permitido para este
	// cliente. nil = sin límite. Se valida al confirmar una venta (ver sales/application/
	// confirm.go) junto con cartera vencida.
	CreditLimit *float64 `json:"credit_limit"`

	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrCustomerNotFound  = errors.New("cliente no encontrado")
	ErrDuplicateCustomer = errors.New("ya existe un cliente con ese número de identificación")
)
