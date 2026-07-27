package nit_test

import (
	"testing"

	"github.com/diegofxm/apidian/internal/catalogs/nit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComputeCheckDigit_RealNIT usa un NIT real con dígito de verificación conocido (RUT real,
// ya usado en pruebas de habilitación contra la DIAN) — no es un caso inventado.
func TestComputeCheckDigit_RealNIT(t *testing.T) {
	got, err := nit.ComputeCheckDigit("6382356")
	require.NoError(t, err)
	assert.Equal(t, "7", got)
}

// TestComputeCheckDigit_FormattedInput confirma que puntos/guiones/espacios se ignoran — el
// mismo NIT con separadores debe dar el mismo dígito.
func TestComputeCheckDigit_FormattedInput(t *testing.T) {
	got, err := nit.ComputeCheckDigit("6.382.356")
	require.NoError(t, err)
	assert.Equal(t, "7", got)
}

// TestComputeCheckDigit_RemainderBranches cubre las dos ramas de la fórmula (resto 0 y resto
// 1, que se devuelven tal cual en vez de 11-resto) — construidos matemáticamente, no
// verificados contra un NIT real (no hace falta: la rama general ya está cubierta arriba).
func TestComputeCheckDigit_RemainderBranches(t *testing.T) {
	// "0" * peso(3) = 0 -> resto 0 -> dígito 0.
	got, err := nit.ComputeCheckDigit("0")
	require.NoError(t, err)
	assert.Equal(t, "0", got)

	// "4" * peso(3) = 12 -> 12 % 11 = 1 -> dígito 1.
	got, err = nit.ComputeCheckDigit("4")
	require.NoError(t, err)
	assert.Equal(t, "1", got)
}

func TestComputeCheckDigit_Invalid(t *testing.T) {
	tests := []string{"", "abc", "900-ABC", "1234567890123456"} // último: más dígitos que pesos definidos
	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			_, err := nit.ComputeCheckDigit(in)
			assert.ErrorIs(t, err, nit.ErrInvalidNumber)
		})
	}
}
