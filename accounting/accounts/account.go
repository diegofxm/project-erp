package accounts

import (
	"time"

	"github.com/google/uuid"
)

type Nature string

const (
	NatureDebit  Nature = "DEBIT"
	NatureCredit Nature = "CREDIT"
)

// Account representa una cuenta del PUC colombiano.
type Account struct {
	ID         uuid.UUID
	Code       string
	Name       string
	ParentCode string
	Level      int
	Category   string
	IsPosting  bool
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Nature devuelve si la cuenta aumenta con débito o con crédito, según su categoría PUC.
// Los valores de category provienen del CSV generado por docs/reference/puc/main.go.
func (a Account) Nature() Nature {
	switch a.Category {
	case "Activo", "Gasto", "Costo", "Costo de Producción", "Cuenta de Orden Deudora":
		return NatureDebit
	default: // Pasivo, Patrimonio, Ingreso, Cuenta de Orden Acreedora
		return NatureCredit
	}
}
