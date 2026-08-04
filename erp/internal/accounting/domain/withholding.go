package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// WithholdingConcept es un concepto de retención (ReteFuente/ReteIVA/ReteICA) del catálogo
// oficial DIAN — sembrado en seed/withholding_concepts.csv, global (sin CompanyID) igual que
// Account (PUC). Code NO es único por sí solo (ej. "01" existe para JURIDICA y NATURAL) — el
// identificador real de una fila es ID.
type WithholdingConcept struct {
	ID                uuid.UUID
	Code              string
	Name              string
	Type              string // RETEFUENTE | RETEIVA | RETEICA
	RateBP            int    // tarifa en puntos básicos (100 = 1%)
	MinBaseUVT        float64
	AccountPayable    string // cuenta PUC del pasivo por retención (ej. 236540)
	AccountReceivable string
	ApplicableTo      string // JURIDICA | NATURAL | BOTH
	IsActive          bool
}

var ErrWithholdingConceptNotFound = errors.New("concepto de retención no encontrado")

type WithholdingConceptRepository interface {
	List(ctx context.Context) ([]WithholdingConcept, error)
	GetByID(ctx context.Context, id uuid.UUID) (*WithholdingConcept, error)
}

// WithholdingCertificate certifica, ante un tercero, cuánto se le retuvo por concepto en un
// año fiscal — lo que el tercero usa para soportar su propia declaración de renta.
type WithholdingCertificate struct {
	ID            uuid.UUID
	CompanyID     uuid.UUID
	FiscalYear    int
	ThirdPartyNIT string
	ConceptCode   string
	ConceptName   string
	WHType        string
	GrossAmount   float64
	TaxWithheld   float64
	Status        string // "issued"
	IssuedAt      *time.Time
	CreatedAt     time.Time
}

type WithholdingCertificateRepository interface {
	Create(ctx context.Context, c WithholdingCertificate) (*WithholdingCertificate, error)
	List(ctx context.Context, companyID uuid.UUID, year int) ([]WithholdingCertificate, error)
}
