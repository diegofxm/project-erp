package securitycode

import "testing"

// No hay un ejemplo oficial publicado para validar esta fórmula byte a byte (a diferencia
// de CUFE) — esta prueba es de regresión/sanidad: longitud correcta, determinismo, y que
// cada insumo realmente participe del hash.
func TestCompute(t *testing.T) {
	got := Compute("software-id", "1234", "SETP1")
	if len(got) != 96 {
		t.Fatalf("longitud esperada 96 (SHA-384 en hex), got %d: %s", len(got), got)
	}
	if again := Compute("software-id", "1234", "SETP1"); again != got {
		t.Errorf("Compute no es determinístico")
	}
	if other := Compute("software-id", "1234", "SETP2"); other == got {
		t.Error("cambiar documentID debería cambiar el resultado")
	}
	if other := Compute("software-id", "9999", "SETP1"); other == got {
		t.Error("cambiar el PIN debería cambiar el resultado")
	}
}
