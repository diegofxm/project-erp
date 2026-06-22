package builder

import (
	"fmt"

	"github.com/diegofxm/cofacture/domain"
)

// formatAmount renderiza centavos como el string decimal de 2 cifras que exige la DIAN.
func formatAmount(cents int64) string {
	return domain.FormatCents(cents)
}

// formatQuantity renderiza una cantidad con 6 decimales, como exige el anexo técnico.
func formatQuantity(q float64) string {
	return fmt.Sprintf("%.6f", q)
}

// formatPercent renderiza un porcentaje de impuesto con 2 decimales.
func formatPercent(p float64) string {
	return fmt.Sprintf("%.2f", p)
}
