package settings

import (
	"time"

	"github.com/google/uuid"
)

const DefaultBrandColor = "#14345C"
const DefaultPricePerDocumentCOP = 150

type IssuerSettings struct {
	IssuerID             uuid.UUID
	BrandColor           string
	PricePerDocumentCOP  int // precio por documento emitido en pesos colombianos
	UpdatedAt            time.Time
}
