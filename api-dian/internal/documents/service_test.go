package documents_test

import (
	"context"
	"testing"

	"github.com/diegofxm/api-dian/internal/documents"
	"github.com/diegofxm/api-dian/internal/issuers"
	"github.com/diegofxm/api-dian/internal/numbering"
	"github.com/diegofxm/ubl21dian/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIssuerPort y fakeNumberingPort son dobles mínimos — documents.Service no necesita un
// *issuers.Service ni un *numbering.Service reales para probarse, solo lo que declaran
// IssuerPort/NumberingPort (mismo patrón que core-bank/internal/transfers).

type fakeIssuerPort struct {
	issuer *issuers.Issuer
}

func (f *fakeIssuerPort) GetIssuer(_ context.Context, _ uuid.UUID) (*issuers.Issuer, error) {
	if f.issuer == nil {
		return nil, issuers.ErrIssuerNotFound
	}
	return f.issuer, nil
}

type fakeNumberingPort struct {
	nr   *numbering.NumberingRange
	next int64
}

func (f *fakeNumberingPort) GetRange(_ context.Context, _ uuid.UUID) (*numbering.NumberingRange, error) {
	if f.nr == nil {
		return nil, numbering.ErrRangeNotFound
	}
	return f.nr, nil
}

func (f *fakeNumberingPort) ClaimNext(_ context.Context, _ uuid.UUID) (int64, error) {
	f.next++
	return f.next, nil
}

func testIssuer() *issuers.Issuer {
	cert, pwd := selfSignedTestP12()
	return &issuers.Issuer{
		ID:                     uuid.New(),
		NIT:                    "900373076",
		CheckDigit:             "1",
		BusinessName:           "Empresa de Prueba S.A.S.",
		IdentificationTypeCode: "31",
		DepartmentCode:         "11",
		MunicipalityCode:       "11001",
		AddressLine:            "Calle 1 # 2-3",
		Email:                  "facturacion@empresa.test",
		Phone:                  "3001234567",
		// Producción a propósito: es el único ambiente que finalizeAndSend nunca envía por
		// red (ver service.go) — desde que existe SendBillSync, habilitación SIEMPRE intenta
		// enviar de verdad (con o sin TestSetID), y estas son pruebas unitarias sin red.
		Environment:         issuers.EnvironmentProduccion,
		EntityTypeCode:      "1",
		TaxSchemeCode:       "ZZ",
		TaxSchemeName:       "No aplica",
		SoftwareID:          "software-id-de-prueba",
		SoftwarePIN:         "12345",
		Certificate:         cert,
		CertificatePassword: pwd,
	}
}

func testNumberingRange(issuerID uuid.UUID) *numbering.NumberingRange {
	rangeTo := int64(5000)
	return &numbering.NumberingRange{
		ID:                   uuid.New(),
		IssuerID:             issuerID,
		DianDocumentTypeCode: "01",
		Prefix:               "SETP",
		ResolutionNumber:     "18760000001",
		RangeFrom:            1,
		RangeTo:              &rangeTo,
		TechnicalKey:         "clave-tecnica-de-prueba",
		Environment:          numbering.EnvironmentHabilitacion,
	}
}

func testRequest(issuerID, rangeID uuid.UUID) documents.IssueInvoiceRequest {
	return documents.IssueInvoiceRequest{
		IssuerID:         issuerID,
		NumberingRangeID: rangeID,
		Customer: domain.Party{
			Identification: domain.Identification{Number: "222222222222", TypeCode: "13"},
			Name:           "Consumidor Final",
		},
		Lines: []domain.Line{{
			Description:        "Servicio de prueba",
			Quantity:           1,
			UnitCode:           "94",
			LineExtensionCents: 10000,
			UnitPriceCents:     10000,
			Taxes: []domain.Tax{
				{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
			},
		}},
	}
}

func TestIssueInvoice_BuildsSignsAndPersists(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)

	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
	)

	doc, err := svc.IssueInvoice(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	assert.Equal(t, "SETP", doc.Prefix)
	assert.Equal(t, int64(1), doc.Number)
	assert.NotEmpty(t, doc.DocumentKey, "debería tener CUFE calculado")
	assert.NotEmpty(t, doc.QRURL)
	assert.NotEmpty(t, doc.SignedXML)
	assert.Contains(t, doc.SignedXML, "<ds:Signature", "el XML persistido debe estar firmado")
	// iss.Environment es Producción en este fixture → no se intenta enviar, queda en "built".
	assert.Equal(t, documents.StatusBuilt, doc.Status)
	assert.Equal(t, int64(10000), doc.Totals.LineExtensionCents)
	assert.Equal(t, int64(10000), doc.Totals.PayableCents)
}

func TestIssueInvoice_ClaimsSequentialNumbers(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
	)

	first, err := svc.IssueInvoice(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)
	second, err := svc.IssueInvoice(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	assert.Equal(t, int64(1), first.Number)
	assert.Equal(t, int64(2), second.Number)
	assert.NotEqual(t, first.DocumentKey, second.DocumentKey)
}

func TestIssueInvoice_WrongDocumentType(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	nr.DianDocumentTypeCode = "91" // rango de Nota Crédito, no de Factura

	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
	)

	_, err := svc.IssueInvoice(context.Background(), testRequest(iss.ID, nr.ID))
	assert.ErrorIs(t, err, documents.ErrWrongDocumentType)
}

func TestIssueInvoice_Validations(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
	)

	tests := []struct {
		name    string
		mutate  func(*documents.IssueInvoiceRequest)
		wantErr error
	}{
		{"sin emisor", func(r *documents.IssueInvoiceRequest) { r.IssuerID = uuid.Nil }, documents.ErrMissingIssuer},
		{"sin rango", func(r *documents.IssueInvoiceRequest) { r.NumberingRangeID = uuid.Nil }, documents.ErrMissingNumberingRange},
		{"sin lineas", func(r *documents.IssueInvoiceRequest) { r.Lines = nil }, documents.ErrEmptyLines},
		{"sin adquiriente", func(r *documents.IssueInvoiceRequest) { r.Customer = domain.Party{} }, documents.ErrMissingCustomer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testRequest(iss.ID, nr.ID)
			tt.mutate(&req)
			_, err := svc.IssueInvoice(context.Background(), req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// ── Fase 2.6: CreditNote/DebitNote ──────────────────────────────────────────────────────────

func testNoteRequest(issuerID, rangeID uuid.UUID) documents.IssueNoteRequest {
	return documents.IssueNoteRequest{
		IssuerID:         issuerID,
		NumberingRangeID: rangeID,
		Customer: domain.Party{
			Identification: domain.Identification{Number: "222222222222", TypeCode: "13"},
			Name:           "Consumidor Final",
		},
		Lines: []domain.Line{{
			Description:        "Anulación de servicio de prueba",
			Quantity:           1,
			UnitCode:           "94",
			LineExtensionCents: 10000,
			UnitPriceCents:     10000,
			Taxes: []domain.Tax{
				{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
			},
		}},
		BillingReference: documents.BillingReferenceInput{
			Prefix:    "SETP",
			Number:    "990068706",
			CUFE:      "853657dcf2841c55c04338b24cc4db9dfbf87042f1ce1798e53f7b1f0502d00df9bd3f371dea47b02766424976d60ba2",
			IssueDate: "2026-06-20",
		},
		DiscrepancyResponse: &documents.DiscrepancyResponseInput{
			ReferenceID:  "SETP990068706",
			ResponseCode: "2",
			Description:  "Anulación de factura electrónica",
		},
	}
}

func creditNoteRangeFor(issuerID uuid.UUID) *numbering.NumberingRange {
	nr := testNumberingRange(issuerID)
	nr.DianDocumentTypeCode = "91"
	nr.Prefix = "SETPNC"
	return nr
}

func debitNoteRangeFor(issuerID uuid.UUID) *numbering.NumberingRange {
	nr := testNumberingRange(issuerID)
	nr.DianDocumentTypeCode = "92"
	nr.Prefix = "SETPND"
	return nr
}

func TestIssueCreditNote_BuildsSignsAndPersists(t *testing.T) {
	iss := testIssuer()
	nr := creditNoteRangeFor(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
	)

	req := documents.IssueCreditNoteRequest{
		IssueNoteRequest:   testNoteRequest(iss.ID, nr.ID),
		CreditNoteTypeCode: "2",
	}
	doc, err := svc.IssueCreditNote(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "SETPNC", doc.Prefix)
	assert.Equal(t, "91", doc.DianDocumentTypeCode)
	assert.NotEmpty(t, doc.DocumentKey, "debería tener CUDE calculado")
	assert.Contains(t, doc.SignedXML, "<ds:Signature")
	assert.Contains(t, doc.SignedXML, "CreditNote")
	require.NotNil(t, doc.BillingReference)
	assert.Equal(t, "990068706", doc.BillingReference.Number)
	require.NotNil(t, doc.DiscrepancyResponse)
	assert.Equal(t, "2", doc.DiscrepancyResponse.ResponseCode)
	assert.Equal(t, "2", doc.NoteTypeCode)
}

func TestIssueDebitNote_BuildsSignsAndPersists(t *testing.T) {
	iss := testIssuer()
	nr := debitNoteRangeFor(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
	)

	doc, err := svc.IssueDebitNote(context.Background(), testNoteRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	assert.Equal(t, "SETPND", doc.Prefix)
	assert.Equal(t, "92", doc.DianDocumentTypeCode)
	assert.NotEmpty(t, doc.DocumentKey, "debería tener CUDE calculado")
	assert.Contains(t, doc.SignedXML, "<ds:Signature")
	assert.Contains(t, doc.SignedXML, "DebitNote")
	require.NotNil(t, doc.BillingReference)
	assert.Equal(t, "990068706", doc.BillingReference.Number)
	assert.Empty(t, doc.NoteTypeCode, "DebitNote no tiene CreditNoteTypeCode")
}

func TestIssueCreditNote_DifferentCUDEThanInvoiceCUFE(t *testing.T) {
	// El CUDE de la nota usa SoftwarePIN, no la clave técnica del CUFE — deben dar distinto
	// incluso si por error se reutilizaran los mismos datos base.
	iss := testIssuer()
	nr := creditNoteRangeFor(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
	)

	req := documents.IssueCreditNoteRequest{
		IssueNoteRequest:   testNoteRequest(iss.ID, nr.ID),
		CreditNoteTypeCode: "2",
	}
	doc, err := svc.IssueCreditNote(context.Background(), req)
	require.NoError(t, err)
	assert.Len(t, doc.DocumentKey, 96, "CUDE es SHA-384 en hex, 96 caracteres")
}

func TestIssueCreditNote_MissingBillingReference(t *testing.T) {
	iss := testIssuer()
	nr := creditNoteRangeFor(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
	)

	req := documents.IssueCreditNoteRequest{
		IssueNoteRequest:   testNoteRequest(iss.ID, nr.ID),
		CreditNoteTypeCode: "2",
	}
	req.BillingReference = documents.BillingReferenceInput{}

	_, err := svc.IssueCreditNote(context.Background(), req)
	assert.ErrorIs(t, err, documents.ErrMissingBillingReference)
}

func TestIssueDebitNote_WrongDocumentType(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID) // rango de Invoice ("01"), no de DebitNote
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
	)

	_, err := svc.IssueDebitNote(context.Background(), testNoteRequest(iss.ID, nr.ID))
	assert.ErrorIs(t, err, documents.ErrWrongDocumentType)
}
