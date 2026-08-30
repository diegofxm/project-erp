package integrationtest

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/cufe"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/signer"
)

// TestSign_RealCertificate signs a real invoice (data from RUT 6382356) with the real
// certificate and technical key from the certification/testing environment. It is skipped
// by default — it only runs if COFACTURE_TEST_FIXTURES_DIR points to a folder containing
// certificado_cert.pem, certificado_key.pem, and credenciales.txt (same pattern as
// DATABASE_URL in core-bank: the real material is never committed, so the test is skipped
// for anyone else).
func TestSign_RealCertificate(t *testing.T) {
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("COFACTURE_TEST_FIXTURES_DIR not set, skipping real-credential test")
	}

	certPEM, err := os.ReadFile(filepath.Join(dir, "certificado_cert.pem"))
	if err != nil {
		t.Fatalf("read certificate: %v", err)
	}
	keyPEM, err := os.ReadFile(filepath.Join(dir, "certificado_key.pem"))
	if err != nil {
		t.Fatalf("read private key: %v", err)
	}
	cert, key, err := signer.LoadPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("LoadPEM: %v", err)
	}

	env := parseEnvFile(t, filepath.Join(dir, "credenciales.txt"))

	rangeFrom, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_FROM"])
	if err != nil {
		t.Fatalf("parse DIAN_RANGE_DATE_FROM: %v", err)
	}
	rangeTo, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_TO"])
	if err != nil {
		t.Fatalf("parse DIAN_RANGE_DATE_TO: %v", err)
	}

	now := time.Now().In(domain.Bogota)
	inv := domain.Invoice{
		ProfileID:         "DIAN 2.1: Factura Electrónica de Venta",
		EnvironmentCode:   env["DIAN_ENVIRONMENT"],
		OperationTypeCode: "10",
		DocumentTypeCode:  "01",
		HashType:          "CUFE-SHA384",

		Prefix: env["DIAN_PREFIX"],
		Number: "1",

		IssueDate: now.Format("2006-01-02"),
		IssueTime: now.Format("15:04:05-07:00"),

		CurrencyCode: "COP",

		// Real data from RUT 6382356 (Diego Fernando Montoya Vallejo, natural person,
		// not liable for IVA — same pattern as the natural-person case verified
		// in the real invoice FESG27: TaxSchemeCode "ZZ"/"No aplica", TaxLevelCode "R-99-PN").
		Supplier: domain.Party{
			EntityTypeCode: "1",
			Identification: domain.Identification{Number: "6382356", TypeCode: "13"},
			Name:           "DIEGO FERNANDO MONTOYA VALLEJO",
			Address: domain.Address{
				Line:        "CL 13 A 25 26 BRR LAS AMERICAS",
				CityCode:    "76520",
				CityName:    "Palmira",
				StateCode:   "76",
				StateName:   "Valle del Cauca",
				CountryCode: "CO",
				CountryName: "Colombia",
			},
			LiabilityCodes: []string{"R-99-PN"},
			TaxSchemeCode:  "ZZ",
			TaxSchemeName:  "No aplica",
			Phone:          "3186708084",
			Email:          "diegofm.comercial@gmail.com",
		},

		Customer: domain.Party{
			EntityTypeCode: "2",
			Identification: domain.Identification{Number: "222222222222", TypeCode: "13"},
			Name:           "Consumidor Final",
			TaxSchemeCode:  "ZZ",
			TaxSchemeName:  "No aplica",
		},

		PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},

		Totals: domain.Totals{
			LineExtensionCents: 10000,
			TaxExclusiveCents:  10000,
			TaxInclusiveCents:  10000,
			PayableCents:       10000,
		},

		Lines: []domain.Line{{
			Description:        "Servicio de prueba (TestSign_RealCertificate)",
			Quantity:           1,
			UnitCode:           "94",
			LineExtensionCents: 10000,
			UnitPriceCents:     10000,
			ItemCode:           "0001",
			ItemTypeCode:       "999",
			ItemTypeName:       "Estándar de adopción del contribuyente",
		}},

		NumberingRange: domain.NumberingRange{
			AuthorizedCode: env["DIAN_RESOLUTION"],
			Prefix:         env["DIAN_PREFIX"],
			StartNumber:    env["DIAN_RANGE_FROM"],
			EndNumber:      env["DIAN_RANGE_TO"],
			StartDate:      rangeFrom.Format("2006-01-02"),
			EndDate:        rangeTo.Format("2006-01-02"),
		},

		SoftwareProvider: domain.SoftwareProvider{
			ProviderIdentification: domain.Identification{Number: "6382356", TypeCode: "13"},
			SoftwareID:             env["DIAN_SOFTWARE_ID"],
		},
	}

	inv.CUFE = cufe.Compute(inv, env["DIAN_TECHNICAL_KEY"])
	inv.SoftwareSecurityCode = securitycode.Compute(env["DIAN_SOFTWARE_ID"], env["DIAN_PIN"], inv.Prefix+inv.Number)
	inv.QRURL = qr.URL(inv.EnvironmentCode, inv.CUFE)

	doc, err := builder.BuildInvoice(inv)
	if err != nil {
		t.Fatalf("BuildInvoice: %v", err)
	}

	placeholder, err := builder.SignaturePlaceholder(doc)
	if err != nil {
		t.Fatalf("SignaturePlaceholder: %v", err)
	}

	s := signer.New(cert, key)
	if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	verifySignature(t, doc.Root(), &key.PublicKey)

	// Without doc.Indent(): it would rewrite the tree after signing, leaving the saved
	// file different from what was actually canonicalized/signed (see the note in
	// send_testset_test.go, where this caused a real rejection from DIAN).
	out, err := doc.WriteToString()
	if err != nil {
		t.Fatalf("WriteToString: %v", err)
	}
	outputsDir := filepath.Join(dir, "outputs")
	if err := os.MkdirAll(outputsDir, 0o755); err != nil {
		t.Fatalf("create outputs/: %v", err)
	}
	outPath := filepath.Join(outputsDir, "_signed_test_output.xml")
	if err := os.WriteFile(outPath, []byte(out), 0o644); err != nil {
		t.Fatalf("write output: %v", err)
	}
	t.Logf("signed XML written to %s (CUFE=%s)", outPath, inv.CUFE)
}

// canonicalizer matches the one cofacture's own signer package uses (inclusive C14N 1.0) —
// needed here only to independently re-verify a signature after Sign, the same way a real
// verifier would. Ported from cofacture/signer's own unit-test helper of the same name, since
// that file (signer_test.go) stays in cofacture and isn't part of this module.
var canonicalizer = dsig.MakeC14N10RecCanonicalizer()

// verifySignature reconstructs what an independent verifier would do: canonicalize
// ds:SignedInfo and check ds:SignatureValue against the public key.
func verifySignature(t *testing.T, root *etree.Element, pub *rsa.PublicKey) {
	t.Helper()
	sig := root.FindElement("//ds:Signature")
	if sig == nil {
		t.Fatal("ds:Signature was not inserted")
	}

	signedInfo := sig.FindElement("ds:SignedInfo")
	if signedInfo == nil {
		t.Fatal("ds:Signature has no ds:SignedInfo")
	}
	canon, err := canonicalizer.Canonicalize(signedInfo)
	if err != nil {
		t.Fatalf("canonicalize SignedInfo: %v", err)
	}
	hashed := sha256.Sum256(canon)

	sigValueEl := sig.FindElement("ds:SignatureValue")
	if sigValueEl == nil {
		t.Fatal("ds:Signature has no ds:SignatureValue")
	}
	sigValue, err := base64.StdEncoding.DecodeString(sigValueEl.Text())
	if err != nil {
		t.Fatalf("decode SignatureValue: %v", err)
	}

	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], sigValue); err != nil {
		t.Errorf("signature does not verify against the public key: %v", err)
	}

	refs := signedInfo.SelectElements("ds:Reference")
	if len(refs) != 3 {
		t.Fatalf("expected 3 ds:Reference (document, KeyInfo, SignedProperties), got %d", len(refs))
	}
}
