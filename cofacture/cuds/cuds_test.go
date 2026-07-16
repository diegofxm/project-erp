package cuds

import (
	"testing"

	"github.com/diegofxm/cofacture/domain"
)

// TestCompute usa el ejemplo oficial de NC del Anexo Técnico 1.9 (sección 11.4.3/11.4.4)
// para verificar que la fórmula SHA-384(dianhash.Seed(doc, softwarePIN)) es matemáticamente
// correcta. No existe un ejemplo publicado para CUDS en el Anexo con los mismos valores
// de entrada y hash esperado, pero la fórmula es idéntica a la del CUDE: misma función
// dianhash.Seed, mismo SHA-384, mismo lastSeedComponent = softwarePIN. El hash esperado
// coincide con el valor verificado por la DIAN en la sección 11.4.3/11.4.4.
//
// La diferencia semántica con CUDE es que en el Documento Soporte los roles están
// invertidos: Supplier es el tercero no obligado (quien vende al emisor), Customer es el
// emisor (la empresa que adquiere). El hash refleja esa inversión porque los NIT viajan en
// los campos Supplier.Identification.Number y Customer.Identification.Number.
func TestCompute(t *testing.T) {
	doc := domain.Invoice{
		Number:          "8110007871",
		IssueDate:       "2019-01-12",
		IssueTime:       "07:00:00-05:00",
		EnvironmentCode: "1",
		Totals: domain.Totals{
			LineExtensionCents: 500_000,
			PayableCents:       595_000,
		},
		HeaderTaxes: []domain.Tax{
			{TypeCode: "01", TaxAmountCents: 95_000},
		},
		// En DS: Supplier = tercero no obligado (vendor), Customer = emisor.
		Supplier: domain.Party{Identification: domain.Identification{Number: "900373076"}},
		Customer: domain.Party{Identification: domain.Identification{Number: "8355990"}},
	}

	const softwarePIN = "12301"
	// Hash verificado contra la DIAN real (sección 11.4.3/11.4.4 del Anexo Técnico 1.9).
	const want = "907e4444decc9e59c160a2fb3b6659b33dc5b632a5008922b9a62f83f757b1c448e47f5867f2b50dbdb96f48c7681168"

	got := Compute(doc, softwarePIN)
	if got != want {
		t.Errorf("Compute() = %s, want %s", got, want)
	}
	if len(got) != 96 {
		t.Errorf("Compute() length = %d, want 96 (SHA-384 en hex)", len(got))
	}
}
