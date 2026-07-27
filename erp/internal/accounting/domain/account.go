package domain

import (
	"time"

	"github.com/google/uuid"
)

type Nature string

const (
	NatureDebit  Nature = "DEBIT"
	NatureCredit Nature = "CREDIT"
)

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

// Nature devuelve la naturaleza de la cuenta según su categoría PUC colombiano.
// Débito: Activo, Gasto, Costo. Crédito: Pasivo, Patrimonio, Ingreso.
func (a Account) Nature() Nature {
	switch a.Category {
	case "Activo", "Gasto", "Costo", "Costo de Producción", "Cuenta de Orden Deudora":
		return NatureDebit
	default:
		return NatureCredit
	}
}
