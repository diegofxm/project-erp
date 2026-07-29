package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Product es un bien o servicio que la empresa vende o compra.
// Pertenece a un tenant (company_id).
type Product struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`

	// Identificación
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// Clasificación DIAN para facturación electrónica
	UnitMeasureCode      string `json:"unit_measure_code"`
	StandardCode         string `json:"standard_code"`
	StandardCodeType     string `json:"standard_code_type"`
	StandardCodeID       string `json:"standard_code_id"`
	StandardCodeAgencyID string `json:"standard_code_agency_id"`

	// Tipo
	IsService bool `json:"is_service"`

	// Tributación
	TaxSchemeCode string  `json:"tax_scheme_code"`
	TaxSchemeName string  `json:"tax_scheme_name"`
	TaxRate       float64 `json:"tax_rate"`

	// Precio base
	BasePrice float64 `json:"base_price"`

	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrProductNotFound  = errors.New("producto no encontrado")
	ErrDuplicateProduct = errors.New("ya existe un producto con ese código")
)
