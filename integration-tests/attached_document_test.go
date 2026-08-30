package integrationtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/domain"
)

// TestBuildInvoiceAttachedDocument_RealApplicationResponse closes the loop for Phases
// 1.5/1.7/1.8: it uses a real ApplicationResponse (decoded from an actual GetStatusZip
// response captured during Phase 1.7 development, saved under docs/reference) as the
// ValidationResult content, and confirms that BuildInvoiceAttachedDocument accepts it
// without errors. Skipped by default — it depends on a file that is not committed.
func TestBuildInvoiceAttachedDocument_RealApplicationResponse(t *testing.T) {
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("COFACTURE_TEST_FIXTURES_DIR not set, skipping")
	}
	path := filepath.Join(dir, "outputs", "_application_response.xml")
	appResponseXML, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("%s not found (run Phase 1.7/1.8 first to generate it): %v", path, err)
	}

	ad := sampleAttachedDocument()
	ad.ValidationResults = []domain.ValidationResult{
		{
			LineID:                 "1",
			DocumentID:             "SETP990068706",
			DocumentCUFE:           "853657dcf2841c55c04338b24cc4db9dfbf87042f1ce1798e53f7b1f0502d00df9bd3f371dea47b02766424976d60ba2",
			DocumentHashType:       "CUFE-SHA384",
			DocumentIssueDate:      "2026-06-20",
			ApplicationResponseXML: string(appResponseXML),
			ValidatorID:            "Unidad Especial Dirección de Impuestos y Aduanas Nacionales",
			ValidationResultCode:   "00",
			ValidationDate:         "2026-06-20",
			ValidationTime:         "07:30:00-05:00",
		},
	}

	doc, err := builder.BuildInvoiceAttachedDocument(ad)
	if err != nil {
		t.Fatalf("BuildInvoiceAttachedDocument: %v", err)
	}
	out, err := doc.WriteToString()
	if err != nil {
		t.Fatalf("WriteToString: %v", err)
	}
	if len(out) < len(appResponseXML) {
		t.Error("output should at least contain the full embedded ApplicationResponse")
	}
}

// sampleAttachedDocument mirrors cofacture/builder's own test fixture of the same name.
// Ported here because its original defining file (builder/attached_document_test.go) stays in
// cofacture and isn't part of this module.
func sampleAttachedDocument() domain.AttachedDocument {
	return domain.AttachedDocument{
		EnvironmentCode:  "2",
		ID:               "1",
		IssueDate:        "2024-01-20",
		IssueTime:        "10:05:00-05:00",
		ParentDocumentID: "SETP1",

		Sender: domain.AttachedPartyInfo{
			Name:           "MI EMPRESA S.A.S.",
			Identification: domain.Identification{Number: "900123456", TypeCode: "31", VerificationCode: "3"},
			TaxRegimeCode:  "48",
			LiabilityCodes: []string{"O-13", "O-15", "O-23"},
			TaxSchemeCode:  "01",
			TaxSchemeName:  "IVA",
		},
		Receiver: domain.AttachedPartyInfo{
			Name:           "Juan Pérez",
			Identification: domain.Identification{Number: "1234567890", TypeCode: "13"},
			TaxSchemeCode:  "01",
			TaxSchemeName:  "IVA",
		},

		AttachmentXML: "<Invoice>simplified content for this test, not validated here</Invoice>",

		ValidationResults: []domain.ValidationResult{
			{
				LineID:                 "1",
				DocumentID:             "SETP1",
				DocumentCUFE:           "8bb918b19ba22a694f1da11c643b5e9de39adf60311cf179179e9b33381030bcd4c3c3f156c506ed5908f9276f5bd9b4",
				DocumentHashType:       "CUFE-SHA384",
				DocumentIssueDate:      "2024-01-20",
				ApplicationResponseXML: "<ApplicationResponse>simplified content for this test</ApplicationResponse>",
				ValidatorID:            "Unidad Especial Dirección de Impuestos y Aduanas Nacionales",
				ValidationResultCode:   "02",
				ValidationDate:         "2024-01-20",
				ValidationTime:         "10:10:00-05:00",
			},
		},
	}
}
