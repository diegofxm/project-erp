package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Customer es un tercero receptor de documentos electrónicos (comprador, beneficiario).
// Pertenece a un tenant (company_id).
type Customer struct {
	ID        uuid.UUID
	CompanyID uuid.UUID

	// Identificación fiscal DIAN
	IdentificationTypeCode string // 31=NIT, 13=Cédula, 22=CE, 91=NUIP…
	IdentificationNumber   string
	CheckDigit             string // solo para NIT (tipo 31)

	// Nombre
	Name string // razón social o nombre completo

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

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrCustomerNotFound  = errors.New("cliente no encontrado")
	ErrDuplicateCustomer = errors.New("ya existe un cliente con ese número de identificación")
)
