package integrationtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/cude"
	"github.com/diegofxm/cofacture/cuds"
	"github.com/diegofxm/cofacture/cufe"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/event"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
	"github.com/diegofxm/cofacture/zip"
)

// TestFullFlow_Real chains a complete real-DIAN cycle in a single run: a fresh Electronic Sales
// Invoice (FE), a Credit Note (NC) and Debit Note (ND) referencing that same fresh invoice, a
// fresh Support Document (DS), an Adjustment Note (NA) referencing that same fresh DS, and the
// four RADIAN events whose Sender is the acquirer (Acuse de Recibo, Recibido del Bien, Aceptación
// Expresa, Reclamo) referencing the fresh FE. Aceptación Tácita is deliberately excluded — see
// tacit_acceptance_test.go's doc comment for why it cannot be submitted for real in a single run.
// Individual Payroll (NE) is exercised separately by nomina_test.go — it doesn't reference
// anything built here, so it doesn't need to be chained.
//
// Every step here submits via SendBillSync (no TestSetID) — confirmed already working for
// Invoice/Credit Note/Debit Note/Adjustment Note by earlier runs of this suite (see
// adjustment_note_test.go's doc comment) once the Phase 1.7/1.9 test set is closed.
//
// KNOWN OPEN QUESTION for the four acquirer-role RADIAN events: DIAN's own model has the
// ACQUIRER submit these using its own registered software, not the issuer's. Our only real
// registered software/certificate belongs to the issuer (Diego); the acquirer in every document
// here is "Consumidor Final" (NIT 222222222222), a generic placeholder with no real DIAN
// registration of its own. This test submits them anyway, signed with the issuer's certificate,
// with the XML's Sender set to the acquirer per the annex — whether DIAN accepts or rejects that
// combination is exactly what this run is meant to find out; it is not assumed to work.
func TestFullFlow_Real(t *testing.T) {
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("COFACTURE_TEST_FIXTURES_DIR not set, skipping real-DIAN test")
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
	client := soap.New(soap.HabilitacionURL, cert, key)
	s := signer.New(cert, key)

	outputsDir := filepath.Join(dir, "outputs")
	if err := os.MkdirAll(outputsDir, 0o755); err != nil {
		t.Fatalf("create outputs/: %v", err)
	}

	const (
		issuerNIT  = "6382356"
		issuerName = "DIEGO FERNANDO MONTOYA VALLEJO"
	)

	issuerSupplier := domain.Party{
		EntityTypeCode: "1",
		Identification: domain.Identification{Number: issuerNIT, TypeCode: "13"},
		Name:           issuerName,
		Address: domain.Address{
			Line: "CL 13 A 25 26 BRR LAS AMERICAS", CityCode: "76520", CityName: "Palmira",
			StateCode: "76", StateName: "Valle del Cauca", CountryCode: "CO", CountryName: "Colombia",
		},
		LiabilityCodes: []string{"R-99-PN"}, TaxSchemeCode: "ZZ", TaxSchemeName: "No aplica",
		Phone: "3186708084", Email: "diegofm.comercial@gmail.com",
	}
	acquirerCustomer := domain.Party{
		EntityTypeCode: "2",
		Identification: domain.Identification{Number: "222222222222", TypeCode: "13"},
		Name:           "Consumidor Final",
		LiabilityCodes: []string{"R-99-PN"}, TaxSchemeCode: "ZZ", TaxSchemeName: "No aplica",
		Email: "consumidor.final@example.com",
	}
	softwareProvider := domain.SoftwareProvider{
		ProviderIdentification: domain.Identification{Number: issuerNIT, TypeCode: "31", VerificationCode: "7"},
		SoftwareID:             env["DIAN_SOFTWARE_ID"],
	}

	now := time.Now().In(domain.Bogota)

	// submitSync builds the ZIP, sends it via SendBillSync and hard-fails the parent test if
	// DIAN rejects it — the shared tail end of every document type below.
	submitSync := func(t *testing.T, kind zip.DocumentKind, nit, outName string, xmlBytes []byte) *soap.DianResponse {
		t.Helper()
		if err := os.WriteFile(filepath.Join(outputsDir, outName), xmlBytes, 0o644); err != nil {
			t.Fatalf("save local copy of %s: %v", outName, err)
		}
		fileName := zip.DocumentFileName(kind, nit, zip.SoftwarePropioCode, now.Year(), uint32(time.Now().UnixNano()%0xFFFFFFFF))
		zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
		if err != nil {
			t.Fatalf("zip.Build(%s): %v", outName, err)
		}
		result, err := client.SendBillSync(fileName, zipBytes)
		if err != nil {
			t.Fatalf("SendBillSync(%s): %v", outName, err)
		}
		t.Logf("%s -> IsValid=%v StatusCode=%s StatusDescription=%s StatusMessage=%s XmlDocumentKey=%s",
			outName, result.IsValid, result.StatusCode, result.StatusDescription, result.StatusMessage, result.XmlDocumentKey)
		if result.ErrorMessage != nil {
			for _, m := range result.ErrorMessage.Items {
				t.Logf("  %s ErrorMessage: %s", outName, m)
			}
		}
		return result
	}

	// ── FE: fresh Electronic Sales Invoice ──────────────────────────────────────────────────
	var (
		feCUFE   string
		fePrefix = env["DIAN_PREFIX"]
		feNumber string
		feDate   = now.Format("2006-01-02")
	)
	t.Run("FE", func(t *testing.T) {
		rangeFromInt, err := strconv.ParseInt(env["DIAN_RANGE_FROM"], 10, 64)
		if err != nil {
			t.Fatalf("parse DIAN_RANGE_FROM: %v", err)
		}
		feNumber = strconv.FormatInt(rangeFromInt+10+time.Now().Unix()%100000, 10)

		rangeFromDate, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_FROM"])
		if err != nil {
			t.Fatalf("parse DIAN_RANGE_DATE_FROM: %v", err)
		}
		rangeToDate, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_TO"])
		if err != nil {
			t.Fatalf("parse DIAN_RANGE_DATE_TO: %v", err)
		}

		inv := domain.Invoice{
			ProfileID: "DIAN 2.1: Factura Electrónica de Venta", EnvironmentCode: env["DIAN_ENVIRONMENT"],
			OperationTypeCode: "10", DocumentTypeCode: "01", HashType: "CUFE-SHA384",
			Prefix: fePrefix, Number: feNumber,
			IssueDate: feDate, IssueTime: now.Format("15:04:05-07:00"),
			CurrencyCode: "COP",
			Supplier:     issuerSupplier, Customer: acquirerCustomer,
			PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},
			HeaderTaxes: []domain.Tax{
				{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
			},
			Totals: domain.Totals{LineExtensionCents: 10000, TaxExclusiveCents: 10000, TaxInclusiveCents: 10000, PayableCents: 10000},
			Lines: []domain.Line{{
				Description: "Servicio de prueba (TestFullFlow_Real / FE)", Quantity: 1, UnitCode: "94",
				LineExtensionCents: 10000, UnitPriceCents: 10000, ItemCode: "0001", ItemTypeCode: "999",
				ItemTypeName: "Estándar de adopción del contribuyente",
				Taxes: []domain.Tax{
					{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
				},
			}},
			NumberingRange: domain.NumberingRange{
				AuthorizedCode: env["DIAN_RESOLUTION"], Prefix: fePrefix,
				StartNumber: env["DIAN_RANGE_FROM"], EndNumber: env["DIAN_RANGE_TO"],
				StartDate: rangeFromDate.Format("2006-01-02"), EndDate: rangeToDate.Format("2006-01-02"),
			},
			SoftwareProvider: softwareProvider,
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
		if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		xmlBytes, err := doc.WriteToBytes()
		if err != nil {
			t.Fatalf("WriteToBytes: %v", err)
		}

		result := submitSync(t, zip.KindInvoice, issuerNIT, "_flow_fe.xml", xmlBytes)
		if !result.IsValid {
			t.Fatalf("DIAN rejected FE: %s — %s", result.StatusCode, result.StatusDescription)
		}
		feCUFE = inv.CUFE
		t.Logf("FE accepted: %s%s CUFE=%s", fePrefix, feNumber, feCUFE)
	})
	if feCUFE == "" {
		t.Fatal("FE step did not produce a CUFE, aborting the rest of the chain")
	}

	feRef := domain.BillingReference{Prefix: fePrefix, Number: feNumber, CUFE: feCUFE, IssueDate: feDate, HashType: "CUFE-SHA384"}

	// ── NC: Credit Note referencing the fresh FE ────────────────────────────────────────────
	t.Run("NC", func(t *testing.T) {
		rangeFromInt, _ := strconv.ParseInt(env["DIAN_RANGE_FROM"], 10, 64)
		number := strconv.FormatInt(rangeFromInt+11+time.Now().Unix()%100000, 10)

		base := domain.Invoice{
			ProfileID: "DIAN 2.1: Nota Crédito de Factura Electrónica de Venta", EnvironmentCode: env["DIAN_ENVIRONMENT"],
			OperationTypeCode: "20", DocumentTypeCode: "91", HashType: "CUDE-SHA384",
			Prefix: fePrefix, Number: number,
			IssueDate: now.Format("2006-01-02"), IssueTime: now.Format("15:04:05-07:00"),
			CurrencyCode: "COP",
			Supplier:     issuerSupplier, Customer: acquirerCustomer,
			PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},
			HeaderTaxes: []domain.Tax{
				{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
			},
			Totals: domain.Totals{LineExtensionCents: 10000, TaxExclusiveCents: 10000, TaxInclusiveCents: 10000, PayableCents: 10000},
			Lines: []domain.Line{{
				Description: "Anulación de FE " + fePrefix + feNumber + " (TestFullFlow_Real / NC)", Quantity: 1, UnitCode: "94",
				LineExtensionCents: 10000, UnitPriceCents: 10000,
				Taxes: []domain.Tax{
					{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
				},
			}},
			NumberingRange: domain.NumberingRange{
				AuthorizedCode: env["DIAN_RESOLUTION"], Prefix: fePrefix,
				StartNumber: env["DIAN_RANGE_FROM"], EndNumber: env["DIAN_RANGE_TO"],
			},
			SoftwareProvider: softwareProvider,
		}
		cn := domain.CreditNote{
			Invoice: base, CreditNoteTypeCode: "91",
			BillingReference: feRef,
			DiscrepancyResponse: &domain.DiscrepancyResponse{
				ReferenceID: fePrefix + feNumber, ResponseCode: "2", Description: "Anulación de factura electrónica",
			},
		}
		cn.CUFE = cude.Compute(cn.Invoice, env["DIAN_PIN"])
		cn.SoftwareSecurityCode = securitycode.Compute(env["DIAN_SOFTWARE_ID"], env["DIAN_PIN"], cn.Prefix+cn.Number)
		cn.QRURL = qr.URL(cn.EnvironmentCode, cn.CUFE)

		doc, err := builder.BuildCreditNote(cn)
		if err != nil {
			t.Fatalf("BuildCreditNote: %v", err)
		}
		placeholder, err := builder.SignaturePlaceholder(doc)
		if err != nil {
			t.Fatalf("SignaturePlaceholder: %v", err)
		}
		if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		xmlBytes, err := doc.WriteToBytes()
		if err != nil {
			t.Fatalf("WriteToBytes: %v", err)
		}
		submitSync(t, zip.KindCreditNote, issuerNIT, "_flow_nc.xml", xmlBytes)
	})

	// ── ND: Debit Note referencing the fresh FE (never submitted for real before) ───────────
	t.Run("ND", func(t *testing.T) {
		rangeFromInt, _ := strconv.ParseInt(env["DIAN_RANGE_FROM"], 10, 64)
		number := strconv.FormatInt(rangeFromInt+12+time.Now().Unix()%100000, 10)

		// DAJ48 (Technical Annex, DebitNote-specific): cac:Party/cac:PartyLegalEntity/cbc:CompanyID
		// must carry schemeName="31" (NIT) for the issuer — unlike Invoice/CreditNote, which
		// accept Diego's real identification type (cédula, "13") for the same NIT. This mirrors
		// the same "force NIT for this one field" pattern erp already applies for the Support
		// Document's SNO (see erp's supplierAsNIT). VerificationCode "7" matches what this file's
		// softwareProvider and DS Customer already use for this exact NIT.
		ndIssuerSupplier := issuerSupplier
		ndIssuerSupplier.Identification = domain.Identification{Number: issuerNIT, TypeCode: "31", VerificationCode: "7"}

		base := domain.Invoice{
			ProfileID: "DIAN 2.1: Nota Débito de Factura Electrónica de Venta", EnvironmentCode: env["DIAN_ENVIRONMENT"],
			OperationTypeCode: "30", DocumentTypeCode: "92", HashType: "CUDE-SHA384",
			Prefix: fePrefix, Number: number,
			IssueDate: now.Format("2006-01-02"), IssueTime: now.Format("15:04:05-07:00"),
			CurrencyCode: "COP",
			Supplier:     ndIssuerSupplier, Customer: acquirerCustomer,
			PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},
			HeaderTaxes: []domain.Tax{
				{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
			},
			Totals: domain.Totals{LineExtensionCents: 10000, TaxExclusiveCents: 10000, TaxInclusiveCents: 10000, PayableCents: 10000},
			Lines: []domain.Line{{
				Description: "Intereses por mora sobre FE " + fePrefix + feNumber + " (TestFullFlow_Real / ND)", Quantity: 1, UnitCode: "94",
				LineExtensionCents: 10000, UnitPriceCents: 10000,
				Taxes: []domain.Tax{
					{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
				},
			}},
			NumberingRange: domain.NumberingRange{
				AuthorizedCode: env["DIAN_RESOLUTION"], Prefix: fePrefix,
				StartNumber: env["DIAN_RANGE_FROM"], EndNumber: env["DIAN_RANGE_TO"],
			},
			SoftwareProvider: softwareProvider,
		}
		dn := domain.DebitNote{
			Invoice: base, BillingReference: feRef,
			DiscrepancyResponse: &domain.DiscrepancyResponse{
				ReferenceID: fePrefix + feNumber, ResponseCode: "1", Description: "Intereses por mora",
			},
		}
		dn.CUFE = cude.Compute(dn.Invoice, env["DIAN_PIN"])
		dn.SoftwareSecurityCode = securitycode.Compute(env["DIAN_SOFTWARE_ID"], env["DIAN_PIN"], dn.Prefix+dn.Number)
		dn.QRURL = qr.URL(dn.EnvironmentCode, dn.CUFE)

		doc, err := builder.BuildDebitNote(dn)
		if err != nil {
			t.Fatalf("BuildDebitNote: %v", err)
		}
		placeholder, err := builder.SignaturePlaceholder(doc)
		if err != nil {
			t.Fatalf("SignaturePlaceholder: %v", err)
		}
		if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		xmlBytes, err := doc.WriteToBytes()
		if err != nil {
			t.Fatalf("WriteToBytes: %v", err)
		}
		submitSync(t, zip.KindDebitNote, issuerNIT, "_flow_nd.xml", xmlBytes)
	})

	// ── DS: fresh Support Document ───────────────────────────────────────────────────────────
	var (
		dsCUFE   string
		dsNumber string
		dsDate   = now.Format("2006-01-02")
	)
	t.Run("DS", func(t *testing.T) {
		number := fmt.Sprintf("%d", 984000000+((time.Now().UnixNano()/1000)%1000000))
		dsNumber = number

		inv := domain.Invoice{
			ProfileID:       "DIAN 2.1: documento soporte en adquisiciones efectuadas a no obligados a facturar.",
			EnvironmentCode: env["DIAN_ENVIRONMENT"], OperationTypeCode: "10", DocumentTypeCode: "05", HashType: "CUDS-SHA384",
			Prefix: "SEDS", Number: number,
			IssueDate: dsDate, IssueTime: now.Format("15:04:05-07:00"),
			CurrencyCode: "COP",
			// Roles reversed for the Support Document: Supplier = SNO, Customer = ABS (issuer).
			Supplier: domain.Party{
				EntityTypeCode: "2",
				Identification: domain.Identification{Number: "1020304050", TypeCode: "31", VerificationCode: "8"},
				Name:           "María García",
				Address: domain.Address{
					Line: "Vereda El Rosal", CityCode: "05001", CityName: "Medellín", PostalZone: "050001",
					StateCode: "05", StateName: "Antioquia", CountryCode: "CO", CountryName: "Colombia",
				},
				LiabilityCodes: []string{"R-99-PN"}, TaxSchemeCode: "ZZ", TaxSchemeName: "No aplica",
			},
			Customer: domain.Party{
				EntityTypeCode: "2",
				Identification: domain.Identification{Number: issuerNIT, TypeCode: "31", VerificationCode: "7"},
				Name:           "MONTOYA VALLEJO DIEGO FERNANDO",
				Address: domain.Address{
					Line: "CL 13 A 25 26 BRR LAS AMERICAS", CityCode: "76520", CityName: "Palmira",
					StateCode: "76", StateName: "Valle del Cauca", CountryCode: "CO", CountryName: "Colombia",
				},
				LiabilityCodes: []string{"R-99-PN"}, TaxSchemeCode: "ZZ", TaxSchemeName: "No aplica",
			},
			PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},
			HeaderTaxes: []domain.Tax{
				{TaxableAmountCents: 10000, TaxAmountCents: 1900, Percent: 19, TypeCode: "01", TypeName: "IVA"},
			},
			Totals: domain.Totals{LineExtensionCents: 10000, TaxExclusiveCents: 10000, TaxInclusiveCents: 11900, PayableCents: 11900},
			Lines: []domain.Line{{
				Description: "Servicio de prueba DS (TestFullFlow_Real / DS)", Quantity: 1, UnitCode: "94",
				LineExtensionCents: 10000, UnitPriceCents: 10000, ItemCode: "0001", ItemTypeCode: "999",
				ItemTypeName: "Estándar de adopción del contribuyente",
				Taxes: []domain.Tax{
					{TaxableAmountCents: 10000, TaxAmountCents: 1900, Percent: 19, TypeCode: "01", TypeName: "IVA"},
				},
			}},
			NumberingRange: domain.NumberingRange{
				AuthorizedCode: "18760000001", Prefix: "SEDS",
				StartNumber: "984000000", EndNumber: "985000000",
				StartDate: "2026-01-01", EndDate: "2026-12-31",
			},
			SoftwareProvider: softwareProvider,
		}
		inv.CUFE = cuds.Compute(inv, env["DIAN_PIN"])
		inv.SoftwareSecurityCode = securitycode.Compute(env["DIAN_SOFTWARE_ID"], env["DIAN_PIN"], inv.Prefix+inv.Number)
		inv.QRURL = qr.SupportDocumentContent(inv, inv.CUFE, env["DIAN_PIN"])

		doc, err := builder.BuildSupportDocument(inv)
		if err != nil {
			t.Fatalf("BuildSupportDocument: %v", err)
		}
		placeholder, err := builder.SignaturePlaceholder(doc)
		if err != nil {
			t.Fatalf("SignaturePlaceholder: %v", err)
		}
		if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		xmlBytes, err := doc.WriteToBytes()
		if err != nil {
			t.Fatalf("WriteToBytes: %v", err)
		}
		// The issuer's NIT (Customer here) drives the ZIP file name, as in support_document_test.go.
		result := submitSync(t, zip.KindSupportDocument, issuerNIT, "_flow_ds.xml", xmlBytes)
		if !result.IsValid {
			t.Fatalf("DIAN rejected DS: %s — %s", result.StatusCode, result.StatusDescription)
		}
		dsCUFE = inv.CUFE
		t.Logf("DS accepted: SEDS%s CUDS=%s", dsNumber, dsCUFE)
	})
	if dsCUFE == "" {
		t.Log("DS step did not produce a CUDS — skipping NA only, the rest of the chain (RADIAN) does not depend on it")
	}

	// ── NA: Adjustment Note referencing the fresh DS (skipped if DS failed above) ───────────
	if dsCUFE != "" {
		t.Run("NA", func(t *testing.T) {
			number := fmt.Sprintf("%d", 1+((time.Now().UnixNano()/1000)%999999))

			base := domain.Invoice{
				ProfileID:       "DIAN 2.1: Nota de ajuste al documento soporte en adquisiciones efectuadas a sujetos no obligados a expedir factura o documento equivalente",
				EnvironmentCode: env["DIAN_ENVIRONMENT"], OperationTypeCode: "10", DocumentTypeCode: "95", HashType: "CUDS-SHA384",
				Prefix: "NAP", Number: number,
				IssueDate: now.Format("2006-01-02"), IssueTime: now.Format("15:04:05-07:00"),
				CurrencyCode: "COP",
				Supplier: domain.Party{
					EntityTypeCode: "2",
					Identification: domain.Identification{Number: "1234567895", TypeCode: "31", VerificationCode: "9"},
					Name:           "Proveedor Prueba",
					Address: domain.Address{
						Line: "CL 1 2 3", CityCode: "86757", CityName: "San Miguel", PostalZone: "000000",
						StateCode: "86", StateName: "Putumayo", CountryCode: "CO", CountryName: "Colombia",
					},
					LiabilityCodes: []string{"R-99-PN"}, TaxSchemeCode: "ZZ", TaxSchemeName: "No aplica",
				},
				Customer: domain.Party{
					EntityTypeCode: "2",
					Identification: domain.Identification{Number: issuerNIT, TypeCode: "31", VerificationCode: "7"},
					Name:           "MONTOYA VALLEJO DIEGO FERNANDO",
					Address: domain.Address{
						Line: "CL 13 A 25 26 BRR LAS AMERICAS", CityCode: "76520", CityName: "Palmira", PostalZone: "000000",
						StateCode: "76", StateName: "Valle del Cauca", CountryCode: "CO", CountryName: "Colombia",
					},
					LiabilityCodes: []string{"R-99-PN"}, TaxSchemeCode: "ZZ", TaxSchemeName: "No aplica",
				},
				PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},
				HeaderTaxes: []domain.Tax{
					{TaxableAmountCents: 10000, TaxAmountCents: 1900, Percent: 19, TypeCode: "01", TypeName: "IVA"},
				},
				Totals: domain.Totals{LineExtensionCents: 10000, TaxExclusiveCents: 10000, TaxInclusiveCents: 11900, PayableCents: 11900},
				Lines: []domain.Line{{
					Description: "Ajuste al DS SEDS" + dsNumber + " (TestFullFlow_Real / NA)", Quantity: 1, UnitCode: "94",
					LineExtensionCents: 10000, UnitPriceCents: 10000, ItemCode: "0001", ItemTypeCode: "999",
					ItemTypeName: "Estándar de adopción del contribuyente",
					Taxes: []domain.Tax{
						{TaxableAmountCents: 10000, TaxAmountCents: 1900, Percent: 19, TypeCode: "01", TypeName: "IVA"},
					},
				}},
				SoftwareProvider: softwareProvider,
			}
			an := domain.AdjustmentNote{
				Invoice: base,
				BillingReference: domain.BillingReference{
					Prefix: "SEDS", Number: dsNumber, CUFE: dsCUFE, IssueDate: dsDate, HashType: "CUDS-SHA384",
				},
				DiscrepancyResponse: &domain.DiscrepancyResponse{
					ReferenceID: "SEDS" + dsNumber, ResponseCode: "2", Description: "Ajuste al documento soporte",
				},
			}
			an.CUFE = cuds.Compute(an.Invoice, env["DIAN_PIN"])
			an.SoftwareSecurityCode = securitycode.Compute(env["DIAN_SOFTWARE_ID"], env["DIAN_PIN"], an.Prefix+an.Number)
			an.QRURL = qr.AdjustmentNoteContent(an.Invoice, an.CUFE, env["DIAN_PIN"])

			doc, err := builder.BuildAdjustmentNote(an)
			if err != nil {
				t.Fatalf("BuildAdjustmentNote: %v", err)
			}
			placeholder, err := builder.SignaturePlaceholder(doc)
			if err != nil {
				t.Fatalf("SignaturePlaceholder: %v", err)
			}
			if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
				t.Fatalf("Sign: %v", err)
			}
			xmlBytes, err := doc.WriteToBytes()
			if err != nil {
				t.Fatalf("WriteToBytes: %v", err)
			}
			submitSync(t, zip.KindAdjustmentNote, issuerNIT, "_flow_na.xml", xmlBytes)
		})
	}

	// ── RADIAN: the four acquirer-role events referencing the fresh FE ──────────────────────
	// Sender = acquirer, Receiver = issuer for all four (per the annex); see the doc comment on
	// this test for the open question about submitting them with the issuer's own credentials.
	buildEventBase := func(id string) domain.Event {
		return domain.Event{
			EnvironmentCode: env["DIAN_ENVIRONMENT"],
			ID:              id,
			IssueDate:       now.Format("2006-01-02"),
			IssueTime:       now.Format("15:04:05-07:00"),
			DocumentReference: domain.EventDocumentReference{
				Prefix: fePrefix, Number: feNumber, CUFE: feCUFE, HashType: "CUFE-SHA384", DocumentTypeCode: "01",
			},
			Sender:           domain.EventParty{Name: "Consumidor Final", Identification: domain.Identification{Number: "222222222222", TypeCode: "13"}, TaxSchemeCode: "ZZ", TaxSchemeName: "No aplica"},
			Receiver:         domain.EventParty{Name: issuerName, Identification: domain.Identification{Number: issuerNIT, TypeCode: "31", VerificationCode: "7"}, TaxSchemeCode: "ZZ", TaxSchemeName: "No aplica"},
			SoftwareProvider: softwareProvider,
		}
	}
	finishEvent := func(t *testing.T, ev *domain.Event, responseCode string) {
		t.Helper()
		ev.CUDE = event.Compute(ev.ID, ev.IssueDate, ev.IssueTime,
			ev.Sender.Identification.Number, ev.Receiver.Identification.Number,
			responseCode, ev.DocumentReference.Prefix+ev.DocumentReference.Number,
			ev.DocumentReference.DocumentTypeCode, env["DIAN_PIN"])
		ev.SoftwareSecurityCode = securitycode.Compute(env["DIAN_SOFTWARE_ID"], env["DIAN_PIN"], ev.ID)
		ev.QRURL = qr.URL(ev.EnvironmentCode, ev.DocumentReference.CUFE)
	}
	// submitEvent zips the signed event and sends it via SendEventUpdateStatus — NOT
	// SendBillSync, which is only for Invoice/CreditNote/DebitNote/SupportDocument/
	// AdjustmentNote. Hard-fails the subtest if DIAN rejects it.
	submitEvent := func(t *testing.T, outName string, xmlBytes []byte) *soap.DianResponse {
		t.Helper()
		if err := os.WriteFile(filepath.Join(outputsDir, outName), xmlBytes, 0o644); err != nil {
			t.Fatalf("save local copy of %s: %v", outName, err)
		}
		fileName := zip.DocumentFileName(zip.KindApplicationResponse, issuerNIT, zip.SoftwarePropioCode, now.Year(), uint32(time.Now().UnixNano()%0xFFFFFFFF))
		zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
		if err != nil {
			t.Fatalf("zip.Build(%s): %v", outName, err)
		}
		result, err := client.SendEventUpdateStatus(zipBytes)
		if err != nil {
			t.Fatalf("SendEventUpdateStatus(%s): %v", outName, err)
		}
		t.Logf("%s -> IsValid=%v StatusCode=%s StatusDescription=%s StatusMessage=%s",
			outName, result.IsValid, result.StatusCode, result.StatusDescription, result.StatusMessage)
		if result.ErrorMessage != nil {
			for _, m := range result.ErrorMessage.Items {
				t.Logf("  %s ErrorMessage: %s", outName, m)
			}
		}
		return result
	}

	t.Run("RADIAN_AcuseRecibo", func(t *testing.T) {
		ev := buildEventBase("1")
		ev.ReceiverPerson = &domain.EventReceiverPerson{
			Identification: domain.Identification{Number: "222222222222", TypeCode: "13"},
			FirstName:      "Consumidor", FamilyName: "Final", JobTitle: "N/A",
		}
		finishEvent(t, &ev, event.ResponseCodeAcuseRecibo)

		doc, err := builder.BuildAcuseRecibo(ev)
		if err != nil {
			t.Fatalf("BuildAcuseRecibo: %v", err)
		}
		placeholder, err := builder.SignaturePlaceholder(doc)
		if err != nil {
			t.Fatalf("SignaturePlaceholder: %v", err)
		}
		if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		xmlBytes, err := doc.WriteToBytes()
		if err != nil {
			t.Fatalf("WriteToBytes: %v", err)
		}
		submitEvent(t, "_flow_ar_acuse.xml", xmlBytes)
	})

	t.Run("RADIAN_RecibidoBien", func(t *testing.T) {
		ev := buildEventBase("2")
		finishEvent(t, &ev, event.ResponseCodeRecibidoBien)

		doc, err := builder.BuildRecibidoBien(ev)
		if err != nil {
			t.Fatalf("BuildRecibidoBien: %v", err)
		}
		placeholder, err := builder.SignaturePlaceholder(doc)
		if err != nil {
			t.Fatalf("SignaturePlaceholder: %v", err)
		}
		if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		xmlBytes, err := doc.WriteToBytes()
		if err != nil {
			t.Fatalf("WriteToBytes: %v", err)
		}
		submitEvent(t, "_flow_ar_recibido.xml", xmlBytes)
	})

	t.Run("RADIAN_AceptacionExpresa", func(t *testing.T) {
		ev := buildEventBase("3")
		finishEvent(t, &ev, event.ResponseCodeAceptacionExpresa)

		doc, err := builder.BuildAceptacionExpresa(ev)
		if err != nil {
			t.Fatalf("BuildAceptacionExpresa: %v", err)
		}
		placeholder, err := builder.SignaturePlaceholder(doc)
		if err != nil {
			t.Fatalf("SignaturePlaceholder: %v", err)
		}
		if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		xmlBytes, err := doc.WriteToBytes()
		if err != nil {
			t.Fatalf("WriteToBytes: %v", err)
		}
		submitEvent(t, "_flow_ar_aceptacion.xml", xmlBytes)
	})

	t.Run("RADIAN_Reclamo", func(t *testing.T) {
		r := domain.Reclamo{
			Event:           buildEventBase("4"),
			RejectionListID: "2",
			RejectionName:   "Reclamo",
		}
		finishEvent(t, &r.Event, event.ResponseCodeReclamo)

		doc, err := builder.BuildReclamo(r)
		if err != nil {
			t.Fatalf("BuildReclamo: %v", err)
		}
		placeholder, err := builder.SignaturePlaceholder(doc)
		if err != nil {
			t.Fatalf("SignaturePlaceholder: %v", err)
		}
		if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		xmlBytes, err := doc.WriteToBytes()
		if err != nil {
			t.Fatalf("WriteToBytes: %v", err)
		}
		submitEvent(t, "_flow_ar_reclamo.xml", xmlBytes)
	})
}
