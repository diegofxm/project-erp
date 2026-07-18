package cuds

import (
	"testing"

	"github.com/diegofxm/cofacture/domain"
)

// TestCompute verifica la fórmula del CUDS: SHA-384 sobre
// Prefix+Number+Date+Time+ValDS+CodImp+ValImp+ValTot+NitSNO+NitABS+PIN+Env.
// El hash esperado fue derivado de la implementación verificada en TestCompute_EjemplosOficial_DS
// y sirve como regression guard para la fórmula.
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
	const want = "3f71b4f6e9b334028ca4cb00aaf4d523e2551bce3175564fdfb53ff90f105c63cf45bef801b0258ce146a526a0da0fef"

	got := Compute(doc, softwarePIN)
	if got != want {
		t.Errorf("Compute() = %s, want %s", got, want)
	}
	if len(got) != 96 {
		t.Errorf("Compute() length = %d, want 96 (SHA-384 en hex)", len(got))
	}
}

// TestCompute_Ejemplo_Oficial_DS verifica la fórmula del CUDS con el ejemplo oficial de la
// Caja de Herramientas DS v1.1 (DocumentoSoporte-OperacionConResidente.xml). Este ejemplo
// tiene un CUDS real calculado por la DIAN, lo que nos permite verificar empíricamente el
// orden correcto de los NITs en la cadena de hash.
//
// Valores del ejemplo:
//   - Prefix+Number: DS236000000
//   - IssueDate: 2022-02-18, IssueTime: 13:34:59-05:00
//   - LineExtension: 3899000.00, IVA: 322430.00, INC: 110100.00, ICA: 0.00, Payable: 4152176.00
//   - SNO NIT (AccountingSupplierParty): "1020"
//   - ABS NIT (AccountingCustomerParty): "800197268"
//   - PIN: "12345", EnvCode: "2"
//   - CUDS esperado: c96a728f4453822bfc69b94253880d21d29dd1a9424444da07610799c203506d33fa4f16830dbd6ee0febb4711bfa23a
func TestCompute_EjemplosOficial_DS(t *testing.T) {
	doc := domain.Invoice{
		Prefix:          "DS",
		Number:          "236000000",
		IssueDate:       "2022-02-18",
		IssueTime:       "13:34:59-05:00",
		EnvironmentCode: "2",
		Totals: domain.Totals{
			LineExtensionCents: 389_900_000, // 3899000.00
			PayableCents:       415_217_600, // 4152176.00
		},
		// Ambos TaxSubtotal tienen TypeCode="01" (IVA), pero solo el primero entra en el CUDS
		// (igual que en el QR: un solo par CodImp+ValImp). El segundo se ignora.
		HeaderTaxes: []domain.Tax{
			{TypeCode: "01", TaxAmountCents: 32_243_000}, // IVA 19% = 322430.00
			{TypeCode: "01", TaxAmountCents: 11_010_000}, // IVA 5%  = 110100.00
		},
		// Supplier = SNO (no obligado, quien vende al emisor)
		Supplier: domain.Party{Identification: domain.Identification{Number: "1020"}},
		// Customer = ABS (empresa emisora del DS)
		Customer: domain.Party{Identification: domain.Identification{Number: "800197268"}},
	}

	const softwarePIN = "12345"
	// CUDS del XML oficial. Verificado que el primer TaxSubtotal (IVA 19%, 322430.00) es
	// el único que entra en la fórmula — el segundo (IVA 5%, 110100.00) no se acumula.
	const want = "c96a728f4453822bfc69b94253880d21d29dd1a9424444da07610799c203506d33fa4f16830dbd6ee0febb4711bfa23a"

	if got := Compute(doc, softwarePIN); got != want {
		t.Errorf("Compute() = %s\n\t\twant %s", got, want)
	}
}
