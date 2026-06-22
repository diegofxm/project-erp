package cude

import (
	"testing"

	"github.com/diegofxm/cofacture/domain"
)

// TestCompute_CreditNote usa el ejemplo oficial de la sección 11.4.3/11.4.4 del Anexo
// Técnico 1.9 — mismos valores de entrada y mismo hash esperado que publica la DIAN.
func TestCompute_CreditNote(t *testing.T) {
	note := domain.Invoice{
		Number:          "8110007871",
		IssueDate:       "2019-01-12",
		IssueTime:       "07:00:00-05:00",
		EnvironmentCode: "1",
		Totals: domain.Totals{
			LineExtensionCents: 500_000, // 5000.00
			PayableCents:       595_000, // 5950.00
		},
		HeaderTaxes: []domain.Tax{
			{TypeCode: "01", TaxAmountCents: 95_000}, // IVA 950.00
		},
		Supplier: domain.Party{Identification: domain.Identification{Number: "900373076"}},
		Customer: domain.Party{Identification: domain.Identification{Number: "8355990"}},
	}

	const softwarePIN = "12301"
	const want = "907e4444decc9e59c160a2fb3b6659b33dc5b632a5008922b9a62f83f757b1c448e47f5867f2b50dbdb96f48c7681168"

	if got := Compute(note, softwarePIN); got != want {
		t.Errorf("Compute() = %s, want %s", got, want)
	}
}

// TestCompute_DebitNote usa los datos de entrada del ejemplo oficial de la sección
// 11.4.5/11.4.6 — pero NO el hash que el anexo dice que da.
//
// El anexo tiene un error de transcripción en este ejemplo puntual: publica la cadena de
// composición "ND10012019-01-1810:58:00-05:0030000.00010.00042400.00030.0032400.009001972
// 6410254102102012" y dice que su SHA-384 es "b9483dc2...", pero el SHA-384 real de esa
// cadena (verificado aparte, fuera de este paquete, con sha512.Sum384 directo) es
// "3fa73a86...". El campo a campo (XPath de la sección 11.4.6) y la fórmula coinciden
// exactamente con el CUFE y con el ejemplo de Nota Crédito de la sección 11.4.3/11.4.4, que
// sí reproduce su propio hash correctamente — por eso se confía en la fórmula y se usa aquí
// el hash matemáticamente correcto, no el publicado.
func TestCompute_DebitNote(t *testing.T) {
	note := domain.Invoice{
		Number:          "ND1001",
		IssueDate:       "2019-01-18",
		IssueTime:       "10:58:00-05:00",
		EnvironmentCode: "2",
		Totals: domain.Totals{
			LineExtensionCents: 3_000_000, // 30000.00
			PayableCents:       3_240_000, // 32400.00
		},
		HeaderTaxes: []domain.Tax{
			{TypeCode: "04", TaxAmountCents: 240_000}, // INC 2400.00
		},
		Supplier: domain.Party{Identification: domain.Identification{Number: "900197264"}},
		Customer: domain.Party{Identification: domain.Identification{Number: "10254102"}},
	}

	const softwarePIN = "10201"
	const want = "3fa73a86d57d9341c536afde1f85c4efd9d4591c2c22bce4dfb0e6b0d2e83b8f047a8bde7098292e9d2493e60d1c31da"

	if got := Compute(note, softwarePIN); got != want {
		t.Errorf("Compute() = %s, want %s", got, want)
	}
}
