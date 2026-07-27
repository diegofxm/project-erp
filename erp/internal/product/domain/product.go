package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Product es un bien o servicio que la empresa vende o compra.
// Pertenece a un tenant (company_id).
type Product struct {
	ID        uuid.UUID
	CompanyID uuid.UUID

	// Identificación
	Code string // código interno del producto
	Name string
	Description string

	// Clasificación DIAN para facturación electrónica
	UnitMeasureCode  string // código de unidad de medida (catálogo DIAN)
	StandardCode     string // código UNSPSC, EAN-13, etc.
	StandardCodeType string // "UNSPSC", "EAN13", "PARTNO", "OTHER"

	// Tipo
	IsService bool // true = servicio, false = bien físico

	// Tributación
	TaxSchemeCode string  // "01" IVA, "ZZ" no aplica, "04" INC
	TaxSchemeName string
	TaxRate       float64 // porcentaje: 19.0, 5.0, 0.0

	// Precio base (en la moneda de la empresa, puede sobreescribirse por línea de documento)
	BasePrice float64

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrProductNotFound  = errors.New("producto no encontrado")
	ErrDuplicateProduct = errors.New("ya existe un producto con ese código")
)
