package numbering

import (
	"time"

	"github.com/google/uuid"
)

// Environment es el ambiente DIAN — mismos códigos que issuers.Environment, pero definido
// aquí también a propósito: numbering no depende de issuers (ver mapa de dependencias en
// docs/apidian-architecture.md sección 4.1), aunque ambos los use database.
type Environment string

const (
	EnvironmentProduccion   Environment = "1"
	EnvironmentHabilitacion Environment = "2"
)

// NumberingRange es un rango de numeración autorizado para un emisor y un tipo de documento
// DIAN (Invoice/CreditNote/DebitNote/futuro). RangeTo es nil cuando el tipo de documento no
// tiene un tope impuesto por una resolución de la DIAN (la propia DIAN solo exige resolución
// de numeración para Invoice — CreditNote/DebitNote no tienen un rango autorizado que se
// "agote" de la misma forma, pero igual necesitan un consecutivo que nunca se repita).
//
// CurrentNumber es el último número YA reclamado — el próximo a entregar es
// CurrentNumber+1, nunca CurrentNumber directamente (ver ClaimNext en repository.go).
type NumberingRange struct {
	ID                   uuid.UUID
	IssuerID             uuid.UUID
	DianDocumentTypeCode string // FK dian_document_types: "01"/"91"/"92"
	Prefix               string
	ResolutionNumber     string
	ResolutionDate       time.Time
	RangeFrom            int64
	RangeTo              *int64
	CurrentNumber        int64
	ValidFrom            time.Time
	ValidTo              time.Time
	Environment          Environment

	// TechnicalKey es la "clave técnica" (ClTec) que usa cufe.Compute — solo aplica a
	// Invoice (01); CreditNote/DebitNote usan el PIN del software (cude.Compute), no esto.
	// Texto plano aquí, cifrada en Postgres (igual patrón que issuers.Issuer).
	TechnicalKey string

	// TestSetID solo aplica en habilitación (DIAN_TEST_SET_ID en las pruebas reales de
	// cofacture) — identifica el set de pruebas asignado por SendTestSetAsync.
	TestSetID string

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Exhausted indica si el rango ya entregó todos los números autorizados. Un rango sin tope
// (RangeTo nil) nunca se agota.
func (nr NumberingRange) Exhausted() bool {
	return nr.RangeTo != nil && nr.CurrentNumber >= *nr.RangeTo
}

// Status resume en un solo valor si el rango se puede usar hoy para emitir — la misma regla la
// necesitan tanto el panel de administración (para mostrar una insignia) como el selector de
// la factura (para no ofrecer un rango que ya no sirve), así que vive en un solo lugar en vez
// de reimplementarse en el frontend. Prioridad: una desactivación explícita pesa más que un
// agotamiento o vencimiento (si el usuario ya lo borró, no importa por qué dejó de servir).
func (nr NumberingRange) Status() string {
	switch {
	case !nr.IsActive:
		return "inactive"
	case nr.Exhausted():
		return "exhausted"
	case !nr.ValidTo.IsZero() && time.Now().After(nr.ValidTo):
		return "expired"
	default:
		return "active"
	}
}
