package settings

import (
	"time"

	"github.com/google/uuid"
)

const (
	DefaultBrandColor          = "#14345C"
	DefaultPricePerDocumentCOP = 150
	DefaultAffiliationFeeCOP   = 180000
	DefaultRenewalFeeCOP       = 180000
)

type IssuerSettings struct {
	IssuerID            uuid.UUID
	BrandColor          string
	PricePerDocumentCOP int        // precio por documento emitido (COP)
	AffiliationFeeCOP   int        // tarifa de afiliación inicial pactada con este emisor
	RenewalFeeCOP       int        // tarifa de renovación anual pactada con este emisor
	AffiliatedAt        *time.Time // fecha en que se registró la afiliación (nil = aún no afiliado)
	RenewalDueAt        *time.Time // fecha de vencimiento de la vigencia actual (nil = no aplica)
	UpdatedAt           time.Time
}
