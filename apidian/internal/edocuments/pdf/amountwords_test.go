package pdf

import "testing"

func TestAmountInWords(t *testing.T) {
	tests := []struct {
		pesos int64
		want  string
	}{
		{0, "CERO PESOS"},
		{1, "UNO PESOS"},
		{15, "QUINCE PESOS"},
		{21, "VEINTIUNO PESOS"},
		{100, "CIEN PESOS"},
		{101, "CIENTO UNO PESOS"},
		{500, "QUINIENTOS PESOS"},
		// Mismo monto que produjo el mini-paquete de referencia (docs/reference/maroto) — ancla
		// de verificación contra un resultado ya visto y aceptado.
		{195001, "CIENTO NOVENTA Y CINCO MIL UNO PESOS"},
		{1_000_000, "UN MILLÓN PESOS"},
		{2_500_000, "DOS MILLONES QUINIENTOS MIL PESOS"},
		{1_000_001, "UN MILLÓN UNO PESOS"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := AmountInWords(tt.pesos); got != tt.want {
				t.Errorf("AmountInWords(%d) = %q, want %q", tt.pesos, got, tt.want)
			}
		})
	}
}
