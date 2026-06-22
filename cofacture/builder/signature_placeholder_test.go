package builder

import "testing"

// Estas pruebas confirman que SignaturePlaceholder funciona igual para los tres tipos de
// documento — los tres reutilizan appendUBLExtensions, pero esto deja una alarma temprana
// si algún builder llega a divergir esa estructura.

func TestSignaturePlaceholder_Invoice(t *testing.T) {
	doc, err := BuildInvoice(sampleInvoice())
	if err != nil {
		t.Fatalf("BuildInvoice: %v", err)
	}
	if _, err := SignaturePlaceholder(doc); err != nil {
		t.Errorf("SignaturePlaceholder: %v", err)
	}
}

func TestSignaturePlaceholder_CreditNote(t *testing.T) {
	doc, err := BuildCreditNote(sampleCreditNote())
	if err != nil {
		t.Fatalf("BuildCreditNote: %v", err)
	}
	if _, err := SignaturePlaceholder(doc); err != nil {
		t.Errorf("SignaturePlaceholder: %v", err)
	}
}

func TestSignaturePlaceholder_DebitNote(t *testing.T) {
	doc, err := BuildDebitNote(sampleDebitNote())
	if err != nil {
		t.Fatalf("BuildDebitNote: %v", err)
	}
	if _, err := SignaturePlaceholder(doc); err != nil {
		t.Errorf("SignaturePlaceholder: %v", err)
	}
}
