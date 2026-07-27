package domain

import (
	"time"

	"github.com/google/uuid"
)

// Environment DIAN.
type Environment string

const (
	EnvProduccion   Environment = "1"
	EnvHabilitacion Environment = "2"
)

// NumberingRange es un rango de numeración autorizado por la DIAN.
// CurrentNumber es el último número ya reclamado; el próximo es CurrentNumber+1.
type NumberingRange struct {
	ID                   uuid.UUID
	CompanyID            uuid.UUID
	DianDocumentTypeCode string
	Prefix               string
	ResolutionNumber     string
	ResolutionDate       time.Time
	RangeFrom            int64
	RangeTo              *int64
	CurrentNumber        int64
	ValidFrom            time.Time
	ValidTo              time.Time
	Environment          Environment
	TechnicalKey         string // texto plano en memoria, cifrado en DB
	TestSetID            string
	IsActive             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Exhausted indica si el rango agotó todos los números autorizados.
func (nr NumberingRange) Exhausted() bool {
	return nr.RangeTo != nil && nr.CurrentNumber >= *nr.RangeTo
}

// Status resume si el rango puede usarse para emitir.
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
