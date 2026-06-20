package builder

import (
	"os"
	"testing"

	"github.com/diegofxm/ubl21dian/domain"
)

func sampleCreditNote() domain.CreditNote {
	inv := sampleInvoice()
	inv.OperationTypeCode = "20" // nota crédito que referencia una factura específica
	inv.DocumentTypeCode = "91"
	inv.HashType = "CUDE-SHA384"
	inv.Prefix = "SETPNC"
	inv.Number = "1"

	return domain.CreditNote{
		Invoice:            inv,
		CreditNoteTypeCode: "1", // catálogo de tipo de Nota Crédito (ej. "1" = devolución parcial)
		BillingReference: domain.BillingReference{
			Prefix:    "SETP",
			Number:    "1",
			CUFE:      "8bb918b19ba22a694f1da11c643b5e9de39adf60311cf179179e9b33381030bcd4c3c3f156c506ed5908f9276f5bd9b4",
			IssueDate: "2024-01-20",
		},
		DiscrepancyResponse: &domain.DiscrepancyResponse{
			ReferenceID:  "SETP1",
			ResponseCode: "2",
			Description:  "Anulación de factura electrónica",
		},
	}
}

func TestBuildCreditNote_Golden(t *testing.T) {
	doc, err := BuildCreditNote(sampleCreditNote())
	if err != nil {
		t.Fatalf("BuildCreditNote: %v", err)
	}
	doc.Indent(2)
	got, err := doc.WriteToString()
	if err != nil {
		t.Fatalf("WriteToString: %v", err)
	}

	const goldenPath = "testdata/credit_note_golden.xml"

	if *update {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Skip("golden file regenerado, revisar a mano antes de confiar en él")
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (¿falta correr con -update?): %v", err)
	}
	if got != string(want) {
		t.Errorf("XML generado no coincide con %s\n--- got ---\n%s", goldenPath, got)
	}
}
