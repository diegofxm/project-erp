package forex

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRateNotFound = errors.New("tipo de cambio no encontrado para la fecha")
	ErrInvalidRate  = errors.New("tasa de cambio debe ser mayor a cero")
)

// ExchangeRate es la tasa de cambio para una fecha y par de monedas.
// RateX10000 almacena la tasa multiplicada por 10 000 para evitar float64.
// Ejemplo: 1 USD = 4 200.1234 COP → RateX10000 = 42 001 234.
type ExchangeRate struct {
	ID           uuid.UUID
	Date         time.Time
	FromCurrency string // "USD", "EUR", "GBP"…
	ToCurrency   string // siempre "COP" en esta versión
	RateX10000   int64  // TRM × 10 000
	Source       string // "BANREP" | "MANUAL"
	CreatedAt    time.Time
}

// CopCents convierte centavos de moneda extranjera a centavos COP.
// foreignCents: monto en centavos de la moneda origen (ej. 100 USD = 10 000 USD-centavos).
func (r *ExchangeRate) CopCents(foreignCents int64) int64 {
	return foreignCents * r.RateX10000 / 10_000
}

// SetRateRequest describe la tasa a registrar o actualizar.
type SetRateRequest struct {
	Date         time.Time
	FromCurrency string // código ISO 4217: "USD", "EUR"
	RateX10000   int64  // TRM × 10 000
	Source       string // "BANREP" | "MANUAL"
}

// RevaluationLine es una cuenta con saldo en moneda extranjera y su ajuste calculado.
type RevaluationLine struct {
	AccountID      uuid.UUID
	AccountCode    string
	AccountName    string
	ForeignBalance int64 // saldo neto en centavos de moneda extranjera (+ = débito neto)
	RecordedCOP    int64 // saldo neto en COP tal como está registrado (+ = débito neto)
	NewCOP         int64 // saldo neto en COP al nuevo tipo de cambio
	Diff           int64 // ajuste = NewCOP − RecordedCOP (+= ganancia, −= pérdida)
}

// RevaluationResult resume una corrida de diferencial cambiario.
type RevaluationResult struct {
	Currency       string
	AsOf           time.Time
	RateX10000     int64
	Lines          []RevaluationLine
	TotalDiff      int64      // suma algebraica de Diff; + = ganancia neta, − = pérdida neta
	JournalEntryID *uuid.UUID // nil si no hubo diferencia (no se generó asiento)
}

// revalBalance es el resultado interno de la consulta de saldos para revaluación.
type revalBalance struct {
	AccountID   uuid.UUID
	AccountCode string
	AccountName string
	Foreign     int64 // saldo neto en centavos extranjeros
	COP         int64 // saldo neto en centavos COP registrados
}
