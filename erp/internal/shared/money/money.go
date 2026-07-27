package money

import "fmt"

// Money representa un valor monetario en centavos (enteros) para evitar errores
// de punto flotante. Todas las operaciones aritméticas son exactas.
// Por convención colombiana: 1 COP = 100 centavos (1_000_000 pesos = 100_000_000 centavos).
type Money struct {
	Cents    int64
	Currency string // "COP", "USD", "EUR"
}

func New(cents int64, currency string) Money {
	return Money{Cents: cents, Currency: currency}
}

func COP(cents int64) Money { return New(cents, "COP") }
func USD(cents int64) Money { return New(cents, "USD") }

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("money: no se puede operar %s con %s", m.Currency, other.Currency)
	}
	return Money{Cents: m.Cents + other.Cents, Currency: m.Currency}, nil
}

func (m Money) Sub(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("money: no se puede operar %s con %s", m.Currency, other.Currency)
	}
	return Money{Cents: m.Cents - other.Cents, Currency: m.Currency}, nil
}

// Mul multiplica por un entero — p. ej. precio × cantidad.
func (m Money) Mul(factor int64) Money {
	return Money{Cents: m.Cents * factor, Currency: m.Currency}
}

func (m Money) IsZero() bool     { return m.Cents == 0 }
func (m Money) IsNegative() bool { return m.Cents < 0 }

func (m Money) Abs() Money {
	if m.Cents < 0 {
		return Money{Cents: -m.Cents, Currency: m.Currency}
	}
	return m
}

func (m Money) String() string {
	abs := m.Abs()
	sign := ""
	if m.IsNegative() {
		sign = "-"
	}
	return fmt.Sprintf("%s%s %d.%02d", sign, m.Currency, abs.Cents/100, abs.Cents%100)
}
