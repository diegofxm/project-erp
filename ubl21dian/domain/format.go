package domain

import "fmt"

// FormatCents renderiza centavos como el string decimal de 2 cifras que exige la DIAN
// (sin separador de miles, punto como separador decimal) — usado tanto al serializar XML
// como al componer las cadenas de entrada del CUFE/CUDE, que comparten el mismo formato.
func FormatCents(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	s := fmt.Sprintf("%d.%02d", cents/100, cents%100)
	if neg {
		s = "-" + s
	}
	return s
}
