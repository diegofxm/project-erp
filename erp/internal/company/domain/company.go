// Package company gestiona la entidad raíz del sistema: la empresa (tenant).
// Cada registro de cualquier otro módulo lleva un company_id que apunta aquí.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Environment es el ambiente DIAN: producción o habilitación.
type Environment string

const (
	Produccion   Environment = "1"
	Habilitacion Environment = "2"
)

// Company es la entidad raíz del multi-tenancy.
type Company struct {
	ID           uuid.UUID
	NIT          string
	CheckDigit   string
	BusinessName string
	TradeName    string

	// Identificación y ubicación fiscal
	IdentificationTypeCode string
	DepartmentCode         string
	MunicipalityCode       string
	AddressLine            string
	Email                  string
	Phone                  string

	// Clasificación tributaria DIAN
	Environment                 Environment
	EntityTypeCode              string   // "1" persona jurídica, "2" persona natural
	TaxSchemeCode               string   // "ZZ" no aplica, "01" IVA
	TaxSchemeName               string
	LiabilityCodes              []string // ["O-13", "R-99-PJ"]
	TaxRegimeCode               *string
	IndustryClassificationCodes []string // códigos CIIU
	MerchantRegistrationNumber  *string

	// Credenciales DIAN — Factura Electrónica
	SoftwareID          string
	SoftwarePIN         string // cifrado en DB con AES-256-GCM
	Certificate         []byte
	CertificatePassword string // cifrado en DB

	// Credenciales DIAN — Nómina Electrónica
	NeSoftwareID  string
	NeSoftwarePIN string // cifrado en DB

	// Visual
	Logo            []byte
	LogoContentType string

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrCompanyNotFound = errors.New("empresa no encontrada")
	ErrNITTaken        = errors.New("NIT ya registrado")
)
