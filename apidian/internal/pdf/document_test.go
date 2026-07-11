package pdf

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func sampleInput() InvoiceInput {
	rangeTo := int64(995000000)
	return InvoiceInput{
		IssuerBusinessName: "QA Invoice SAS",
		IssuerNIT:          "900373076",
		IssuerCheckDigit:   "1",
		IssuerAddressLine:  "Calle de prueba 123",
		IssuerCityName:     "Bogotá D.C.",
		IssuerStateName:    "Bogotá D.C.",
		IssuerPhone:        "3001234567",
		IssuerEmail:        "facturacion@qainvoice.test",
		CurrencyCode:       "COP",

		CustomerName:               "Cliente QA",
		CustomerIdentificationType: "Cédula de Ciudadanía",
		CustomerIdentification:     "222222222222",
		CustomerAddressLine:        "Calle del cliente 456",
		CustomerPhone:              "3007654321",
		CustomerEmail:              "cliente@qainvoice.test",

		PaymentTermName:   "Contado",
		PaymentMethodName: "Transferencia Débito Bancaria",

		Lines: []InvoiceLine{
			{ItemCode: "SVC-001", Description: "Servicio de prueba", UnitCode: "94", Quantity: 2, UnitPriceCents: 100000, TaxPercent: 19, TotalCents: 238000},
		},
		Taxes: []InvoiceTax{
			{TypeName: "IVA", Percent: 19, TaxableAmountCents: 200000, TaxAmountCents: 38000},
		},
		LineExtensionCents: 200000,
		TaxAmountCents:     38000,
		PayableCents:       238000,

		ResolutionNumber: "18760000001",
		RangePrefix:      "SETP",
		RangeFrom:        990000000,
		RangeTo:          &rangeTo,
		ValidFrom:        "2026-01-01",
		ValidTo:          "2030-12-31",
	}
}

func TestBuildInvoicePDF_Draft(t *testing.T) {
	in := sampleInput()
	in.IsDraft = true

	bytes_, err := BuildInvoicePDF(in)
	if err != nil {
		t.Fatalf("BuildInvoicePDF() error = %v", err)
	}
	assertLooksLikePDF(t, bytes_)
}

func TestBuildInvoicePDF_Confirmed(t *testing.T) {
	in := sampleInput()
	in.IsDraft = false
	in.Prefix = "SETP"
	in.Number = 990000001
	in.CUFE = "e78c5e1b65bde1bfad47319b965b6c6ff9e7af02354fa188372e53c4fe5f84f8c6a9c394a634f1af881db67500a71709"
	in.QRURL = "https://catalogo-vpfe-hab.dian.gov.co/document/searchqr?documentkey=" + in.CUFE
	in.IssueDate = "2026-06-26 15:30:00"

	got, err := BuildInvoicePDF(in)
	if err != nil {
		t.Fatalf("BuildInvoicePDF() error = %v", err)
	}
	assertLooksLikePDF(t, got)

	// Útil para inspección visual manual — no se versiona (gitignored, ver
	// docs/apidian-architecture.md sección 9.39).
	if os.Getenv("PDF_TEST_WRITE_SAMPLE") != "" {
		_ = os.WriteFile("sample_invoice.pdf", got, 0o644)
	}
}

func TestBuildInvoicePDF_WithLogo(t *testing.T) {
	in := sampleInput()
	in.IsDraft = true
	in.IssuerLogo = onePixelPNG()
	in.IssuerLogoExt = "png"

	got, err := BuildInvoicePDF(in)
	if err != nil {
		t.Fatalf("BuildInvoicePDF() error = %v", err)
	}
	assertLooksLikePDF(t, got)
}

// TestResolutionDisclaimer_NoUpperBound confirma el fix: un rango sin tope (RangeTo nil, ver
// numbering.NumberingRange.RangeTo) no debe inventar un "hasta {prefix}0" — antes de esto se
// vio literalmente "Rango autorizado desde QAPDF1 hasta QAPDF0" en un PDF real generado contra
// un rango de prueba sin range_to.
func TestResolutionDisclaimer_NoUpperBound(t *testing.T) {
	in := sampleInput()
	in.RangeTo = nil
	in.RangePrefix = "QAPDF"
	in.RangeFrom = 1

	got := resolutionDisclaimer(in)
	if want := "Rango autorizado desde QAPDF1."; !bytes.Contains([]byte(got), []byte(want)) {
		t.Errorf("resolutionDisclaimer() = %q, esperaba que contuviera %q", got, want)
	}
	if bytes.Contains([]byte(got), []byte("QAPDF0")) {
		t.Errorf("resolutionDisclaimer() = %q, no debería inventar un tope QAPDF0", got)
	}
}

func TestResolutionDisclaimer_WithUpperBound(t *testing.T) {
	in := sampleInput()

	got := resolutionDisclaimer(in)
	want := "Rango autorizado desde SETP990000000 hasta SETP995000000."
	if !bytes.Contains([]byte(got), []byte(want)) {
		t.Errorf("resolutionDisclaimer() = %q, esperaba que contuviera %q", got, want)
	}
}

func assertLooksLikePDF(t *testing.T, b []byte) {
	t.Helper()
	if len(b) == 0 {
		t.Fatal("BuildInvoicePDF() devolvió bytes vacíos")
	}
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatalf("BuildInvoicePDF() no empieza con %%PDF, primeros bytes: %q", b[:min(20, len(b))])
	}
}

// onePixelPNG genera un PNG 1x1 válido en memoria (vía image/png, no bytes a mano) —
// suficiente para probar la ruta con logo sin depender de un archivo de imagen externo.
func onePixelPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
