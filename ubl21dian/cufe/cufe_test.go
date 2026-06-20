package cufe

import (
	"testing"

	"github.com/diegofxm/ubl21dian/domain"
)

// TestCompute_AnexoTecnicoExample usa el ejemplo oficial de la sección 11.2.1 del Anexo
// Técnico 1.9 (Resolución DIAN 000165/2023) — mismos valores de entrada y mismo hash
// esperado que publica la DIAN, no un caso inventado.
func TestCompute_AnexoTecnicoExample(t *testing.T) {
	inv := domain.Invoice{
		Number:          "323200000129",
		IssueDate:       "2019-01-16",
		IssueTime:       "10:53:10-05:00",
		EnvironmentCode: "1",
		Totals: domain.Totals{
			LineExtensionCents: 150_000_000, // 1500000.00
			PayableCents:       178_500_000, // 1785000.00
		},
		HeaderTaxes: []domain.Tax{
			{TypeCode: "01", TaxAmountCents: 28_500_000}, // IVA 285000.00
		},
		Supplier: domain.Party{Identification: domain.Identification{Number: "700085371"}},
		Customer: domain.Party{Identification: domain.Identification{Number: "800199436"}},
	}

	const technicalKey = "693ff6f2a553c3646a063436fd4dd9ded0311471"
	const want = "8bb918b19ba22a694f1da11c643b5e9de39adf60311cf179179e9b33381030bcd4c3c3f156c506ed5908f9276f5bd9b4"

	got := Compute(inv, technicalKey)
	if got != want {
		t.Errorf("Compute() = %s, want %s", got, want)
	}
}
