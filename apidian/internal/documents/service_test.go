package documents_test

import (
	"context"
	"testing"
	"time"

	"github.com/diegofxm/apidian/internal/customers"
	"github.com/diegofxm/apidian/internal/documents"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/diegofxm/cofacture/domain"
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
	nr       *numbering.NumberingRange
	next     int64
	released []int64 // números pasados a ReleaseIfCurrent, en orden — para que los tests confirmen cuándo se intentó devolver uno
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

// ReleaseIfCurrent reproduce la misma condición atómica que numbering.PostgresRepository: solo
// retrocede si number es de verdad el último entregado por ClaimNext.
func (f *fakeNumberingPort) ReleaseIfCurrent(_ context.Context, _ uuid.UUID, number int64) error {
	f.released = append(f.released, number)
	if f.next == number {
		f.next--
	}
	return nil
}

// fakeCustomerPort es un doble mínimo para el CustomerID opcional (ver model.go) — la
// mayoría de los tests no lo necesitan (CustomerID nil), por eso customer queda nil por
// defecto: nunca se llama GetCustomer si el request no trae CustomerID.
type fakeCustomerPort struct {
	customer *customers.Customer
}

func (f *fakeCustomerPort) GetCustomer(_ context.Context, _ uuid.UUID) (*customers.Customer, error) {
	if f.customer == nil {
		return nil, customers.ErrCustomerNotFound
	}
	return f.customer, nil
}

// fakeCatalogPort acepta cualquier código por defecto — los tests que de verdad necesitan
// probar un rechazo (código inválido) ponen valid (payment_means) o validLiability en false
// explícitamente. Campos separados porque ambos chequeos viven en el mismo validateBase pero
// se necesita poder rechazar uno sin afectar al otro (ver
// TestCreateInvoiceDraft_InvalidLiabilityCode).
type fakeCatalogPort struct {
	valid          bool
	validLiability bool
}

func newFakeCatalogPort() *fakeCatalogPort {
	return &fakeCatalogPort{valid: true, validLiability: true}
}

func (f *fakeCatalogPort) IsValidPaymentTerm(_ context.Context, _ string) (bool, error) {
	return f.valid, nil
}

func (f *fakeCatalogPort) IsValidPaymentMethod(_ context.Context, _ string) (bool, error) {
	return f.valid, nil
}

func (f *fakeCatalogPort) IsValidLiabilityCode(_ context.Context, _ string) (bool, error) {
	return f.validLiability, nil
}

// GetTaxTypeName replica el subconjunto de tax_types que usan estos tests.
func (f *fakeCatalogPort) GetTaxTypeName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"ZZ": "No aplica", "01": "IVA"}
	name, ok := names[code]
	return name, ok, nil
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
		Lines: []documents.LineInput{{
			Description:    "Servicio de prueba",
			Quantity:       1,
			UnitCode:       "94",
			UnitPriceCents: 10000,
			TaxTypeCode:    "01",
			TaxPercent:     0,
		}},
		PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},
	}
}

// issueInvoice crea el borrador y lo confirma de una — equivalente al viejo IssueInvoice de
// una sola llamada, para los tests que solo les importa el resultado final ya firmado.
func issueInvoice(t *testing.T, svc *documents.Service, req documents.IssueInvoiceRequest) *documents.Document {
	t.Helper()
	draft, err := svc.CreateInvoiceDraft(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, documents.StatusDraft, draft.Status)
	confirmed, err := svc.ConfirmDocument(context.Background(), req.IssuerID, draft.ID)
	require.NoError(t, err)
	return confirmed
}

func TestCreateInvoiceDraft_OK(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	draft, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	assert.Equal(t, documents.StatusDraft, draft.Status)
	assert.Zero(t, draft.Number, "un borrador no reclama número")
	assert.Empty(t, draft.DocumentKey, "un borrador no tiene CUFE todavía")
	assert.Empty(t, draft.SignedXML, "un borrador no está firmado todavía")
	assert.Equal(t, int64(10000), draft.Totals.LineExtensionCents, "los totales sí se calculan desde el borrador")
}

func TestConfirmDocument_BuildsSignsAndPersists(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	doc := issueInvoice(t, svc, testRequest(iss.ID, nr.ID))

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

func TestConfirmDocument_ClaimsSequentialNumbers(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	first := issueInvoice(t, svc, testRequest(iss.ID, nr.ID))
	second := issueInvoice(t, svc, testRequest(iss.ID, nr.ID))

	assert.Equal(t, int64(1), first.Number)
	assert.Equal(t, int64(2), second.Number)
	assert.NotEqual(t, first.DocumentKey, second.DocumentKey)
}

// TestConfirmDocument_ClaimLoadFailure_ReleasesNumberForRetry confirma el mecanismo central de
// la sección 9.33: si confirmar falla DESPUÉS de reclamar un número pero ANTES de que el
// documento exista de verdad (acá, un certificado corrupto — mismo camino que un rechazo de la
// DIAN o un error de envío, ver finish() en service.go), el número se devuelve, y el SIGUIENTE
// intento sobre el mismo borrador lo vuelve a reclamar en vez de saltar al siguiente y dejar un
// hueco permanente.
func TestConfirmDocument_ClaimLoadFailure_ReleasesNumberForRetry(t *testing.T) {
	iss := testIssuer()
	validCert, validPwd := iss.Certificate, iss.CertificatePassword
	iss.Certificate = []byte("esto no es un certificado PKCS12 válido")
	nr := testNumberingRange(iss.ID)
	numberingPort := &fakeNumberingPort{nr: nr}
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		numberingPort,
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	draft, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	_, err = svc.ConfirmDocument(context.Background(), iss.ID, draft.ID)
	require.Error(t, err, "el certificado corrupto debe fallar al confirmar")
	assert.Equal(t, []int64{1}, numberingPort.released, "el número 1 ya reclamado debe devolverse")
	assert.Equal(t, int64(0), numberingPort.next, "current_number debe quedar como si nunca se hubiera reclamado")

	stillDraft, err := svc.GetDocument(context.Background(), draft.ID)
	require.NoError(t, err)
	assert.Equal(t, documents.StatusDraft, stillDraft.Status, "el borrador nunca llegó a confirmarse, debe seguir en draft")

	// El usuario "arregla" el certificado y reintenta sobre el MISMO borrador.
	iss.Certificate, iss.CertificatePassword = validCert, validPwd
	confirmed, err := svc.ConfirmDocument(context.Background(), iss.ID, draft.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), confirmed.Number, "debe reclamar de nuevo el número 1, no saltar al 2")
}

func TestConfirmDocument_NotDraft(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	doc := issueInvoice(t, svc, testRequest(iss.ID, nr.ID))

	_, err := svc.ConfirmDocument(context.Background(), iss.ID, doc.ID)
	assert.ErrorIs(t, err, documents.ErrDocumentNotDraft, "confirmar dos veces no debe gastar un segundo número")
}

func TestConfirmDocument_OtherIssuer(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	draft, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	_, err = svc.ConfirmDocument(context.Background(), uuid.New(), draft.ID)
	assert.ErrorIs(t, err, documents.ErrDocumentNotFound)
}

func TestConfirmDocument_IssuerNotReady(t *testing.T) {
	iss := testIssuer()
	iss.SoftwareID, iss.SoftwarePIN, iss.Certificate = "", "", nil // emisor sin configurar todavía
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	draft, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err, "crear el borrador NO debe exigir software/certificado todavía")

	_, err = svc.ConfirmDocument(context.Background(), iss.ID, draft.ID)
	assert.ErrorIs(t, err, documents.ErrIssuerNotReadyToIssue)
}

func TestUpdateInvoiceDraft_OK(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	draft, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	updated := testRequest(iss.ID, nr.ID)
	updated.Lines[0].Description = "Servicio corregido"
	updated.Lines[0].UnitPriceCents = 20000

	got, err := svc.UpdateInvoiceDraft(context.Background(), draft.ID, updated)
	require.NoError(t, err)
	assert.Equal(t, "Servicio corregido", got.Lines[0].Description)
	assert.Equal(t, int64(20000), got.Totals.LineExtensionCents)
}

func TestUpdateInvoiceDraft_NotDraft(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	doc := issueInvoice(t, svc, testRequest(iss.ID, nr.ID))

	_, err := svc.UpdateInvoiceDraft(context.Background(), doc.ID, testRequest(iss.ID, nr.ID))
	assert.ErrorIs(t, err, documents.ErrDocumentNotDraft, "un documento ya confirmado es inmutable")
}

func TestUpdateInvoiceDraft_OtherIssuer(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	draft, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	otroReq := testRequest(uuid.New(), nr.ID)
	_, err = svc.UpdateInvoiceDraft(context.Background(), draft.ID, otroReq)
	assert.ErrorIs(t, err, documents.ErrDocumentNotFound)
}

func TestDeleteDraft_OK(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	draft, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	require.NoError(t, svc.DeleteDraft(context.Background(), iss.ID, draft.ID))
	_, err = svc.GetDocument(context.Background(), draft.ID)
	assert.ErrorIs(t, err, documents.ErrDocumentNotFound)
}

func TestDeleteDraft_NotDraft(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	doc := issueInvoice(t, svc, testRequest(iss.ID, nr.ID))

	err := svc.DeleteDraft(context.Background(), iss.ID, doc.ID)
	assert.ErrorIs(t, err, documents.ErrDocumentNotDraft, "un documento ya confirmado nunca se borra")
}

func TestCreateInvoiceDraft_WrongDocumentType(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	nr.DianDocumentTypeCode = "91" // rango de Nota Crédito, no de Factura

	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	_, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	assert.ErrorIs(t, err, documents.ErrWrongDocumentType, "el tipo se valida desde el borrador, no solo al confirmar")
}

func TestCreateInvoiceDraft_NumberingRangeIssuerMismatch(t *testing.T) {
	iss := testIssuer()
	otroEmisorID := uuid.New() // el rango pertenece a OTRO emisor, no a iss
	nr := testNumberingRange(otroEmisorID)

	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	_, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	assert.ErrorIs(t, err, documents.ErrNumberingRangeIssuerMismatch)
}

func TestCreateInvoiceDraft_WithCustomerID_OK(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	cust := &customers.Customer{ID: uuid.New(), IssuerID: iss.ID}

	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{customer: cust},
		newFakeCatalogPort(),
	)

	req := testRequest(iss.ID, nr.ID)
	req.CustomerID = &cust.ID
	draft, err := svc.CreateInvoiceDraft(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, draft.CustomerID)
	assert.Equal(t, cust.ID, *draft.CustomerID)
}

func TestCreateInvoiceDraft_CustomerIssuerMismatch(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	otroEmisorID := uuid.New()
	cust := &customers.Customer{ID: uuid.New(), IssuerID: otroEmisorID} // cliente de OTRO emisor

	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{customer: cust},
		newFakeCatalogPort(),
	)

	req := testRequest(iss.ID, nr.ID)
	req.CustomerID = &cust.ID
	_, err := svc.CreateInvoiceDraft(context.Background(), req)
	assert.ErrorIs(t, err, documents.ErrCustomerIssuerMismatch)
}

func TestCreateInvoiceDraft_CustomerIDNotFound(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)

	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{}, // sin customer configurado -> GetCustomer siempre falla
		newFakeCatalogPort(),
	)

	req := testRequest(iss.ID, nr.ID)
	missing := uuid.New()
	req.CustomerID = &missing
	_, err := svc.CreateInvoiceDraft(context.Background(), req)
	assert.ErrorIs(t, err, customers.ErrCustomerNotFound)
}

func TestCreateInvoiceDraft_Validations(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
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
		{"sin forma de pago", func(r *documents.IssueInvoiceRequest) { r.PaymentMeans = nil }, documents.ErrMissingPaymentMeans},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testRequest(iss.ID, nr.ID)
			tt.mutate(&req)
			_, err := svc.CreateInvoiceDraft(context.Background(), req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestCreateInvoiceDraft_InvalidPaymentMeans confirma el hallazgo de la auditoría de
// catálogos huérfanos: payment_means vive en JSONB sin FK posible, así que un código que no
// existe en payment_terms/payment_methods se rechaza en el servicio (CatalogPort), no solo al
// confirmar con la DIAN rechazando el documento ya con un número real reclamado.
func TestCreateInvoiceDraft_InvalidPaymentMeans(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		&fakeCatalogPort{valid: false},
	)

	req := testRequest(iss.ID, nr.ID)
	_, err := svc.CreateInvoiceDraft(context.Background(), req)
	// fakeCatalogPort rechaza ambos catálogos cuando valid=false; validateBase comprueba
	// PaymentTerm primero, así que ese es el error que se ve siempre.
	assert.ErrorIs(t, err, documents.ErrInvalidPaymentTerm)
}

// TestCreateInvoiceDraft_InvalidLiabilityCode confirma la misma protección que
// TestCreateInvoiceDraft_InvalidPaymentMeans pero para customer.liability_codes — el otro
// catálogo que tampoco puede tener FK (TEXT[], no JSONB) y que se cerró en la misma sesión.
func TestCreateInvoiceDraft_InvalidLiabilityCode(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		&fakeCatalogPort{valid: true, validLiability: false},
	)

	req := testRequest(iss.ID, nr.ID)
	req.Customer.LiabilityCodes = []string{"CODIGO-INVENTADO"}
	_, err := svc.CreateInvoiceDraft(context.Background(), req)
	assert.ErrorIs(t, err, documents.ErrInvalidLiabilityCode)
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
		Lines: []documents.LineInput{{
			Description:    "Anulación de servicio de prueba",
			Quantity:       1,
			UnitCode:       "94",
			UnitPriceCents: 10000,
			TaxTypeCode:    "01",
			TaxPercent:     0,
		}},
		PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},
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

// issueCreditNote/issueDebitNote: crear el borrador y confirmarlo de una, mismo criterio que
// issueInvoice.
func issueCreditNote(t *testing.T, svc *documents.Service, req documents.IssueCreditNoteRequest) *documents.Document {
	t.Helper()
	draft, err := svc.CreateCreditNoteDraft(context.Background(), req)
	require.NoError(t, err)
	confirmed, err := svc.ConfirmDocument(context.Background(), req.IssuerID, draft.ID)
	require.NoError(t, err)
	return confirmed
}

func issueDebitNote(t *testing.T, svc *documents.Service, req documents.IssueNoteRequest) *documents.Document {
	t.Helper()
	draft, err := svc.CreateDebitNoteDraft(context.Background(), req)
	require.NoError(t, err)
	confirmed, err := svc.ConfirmDocument(context.Background(), req.IssuerID, draft.ID)
	require.NoError(t, err)
	return confirmed
}

func TestIssueCreditNote_BuildsSignsAndPersists(t *testing.T) {
	iss := testIssuer()
	nr := creditNoteRangeFor(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	req := documents.IssueCreditNoteRequest{
		IssueNoteRequest:   testNoteRequest(iss.ID, nr.ID),
		CreditNoteTypeCode: "2",
	}
	doc := issueCreditNote(t, svc, req)

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
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	doc := issueDebitNote(t, svc, testNoteRequest(iss.ID, nr.ID))

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
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	req := documents.IssueCreditNoteRequest{
		IssueNoteRequest:   testNoteRequest(iss.ID, nr.ID),
		CreditNoteTypeCode: "2",
	}
	doc := issueCreditNote(t, svc, req)
	assert.Len(t, doc.DocumentKey, 96, "CUDE es SHA-384 en hex, 96 caracteres")
}

func TestCreateCreditNoteDraft_MissingBillingReference(t *testing.T) {
	iss := testIssuer()
	nr := creditNoteRangeFor(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	req := documents.IssueCreditNoteRequest{
		IssueNoteRequest:   testNoteRequest(iss.ID, nr.ID),
		CreditNoteTypeCode: "2",
	}
	req.BillingReference = documents.BillingReferenceInput{}

	_, err := svc.CreateCreditNoteDraft(context.Background(), req)
	assert.ErrorIs(t, err, documents.ErrMissingBillingReference)
}

func TestCreateDebitNoteDraft_WrongDocumentType(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID) // rango de Invoice ("01"), no de DebitNote
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
	)

	_, err := svc.CreateDebitNoteDraft(context.Background(), testNoteRequest(iss.ID, nr.ID))
	assert.ErrorIs(t, err, documents.ErrWrongDocumentType)
}

// ── Listado ──────────────────────────────────────────────────────────────────────────────────

// seedDocument inserta un documento directamente en repo (sin pasar por CreateInvoiceDraft —
// no hace falta firmar de verdad para probar filtros de listado).
func seedDocument(t *testing.T, repo documents.Repository, issuerID uuid.UUID, docType, status string, issueDate time.Time) {
	t.Helper()
	_, err := repo.Create(context.Background(), documents.Document{
		IssuerID:             issuerID,
		NumberingRangeID:     uuid.New(),
		DianDocumentTypeCode: docType,
		Prefix:               "SETP",
		Number:               1,
		DocumentKey:          "cufe-de-prueba",
		IssueDate:            issueDate,
		IssueTime:            "10:00:00-05:00",
		CurrencyCode:         "COP",
		Customer:             domain.Party{Name: "Consumidor Final"},
		Lines:                []domain.Line{{Description: "x", Quantity: 1}},
		QRURL:                "https://catalogo-vpfe.dian.gov.co/document/searchqr?...",
		SignedXML:            "<xml/>",
		Status:               documents.Status(status),
	})
	require.NoError(t, err)
}

func TestListDocuments_FiltersByIssuer(t *testing.T) {
	repo := documents.NewMemoryRepository()
	svc := documents.New(repo, &fakeIssuerPort{}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort())
	issuerA, issuerB := uuid.New(), uuid.New()

	seedDocument(t, repo, issuerA, "01", "accepted", time.Now())
	seedDocument(t, repo, issuerA, "91", "built", time.Now())
	seedDocument(t, repo, issuerB, "01", "accepted", time.Now())

	got, err := svc.ListDocuments(context.Background(), issuerA, documents.ListFilter{})
	require.NoError(t, err)
	assert.Len(t, got, 2, "solo los documentos del emisor A")
}

func TestListDocuments_FiltersByTypeAndStatus(t *testing.T) {
	repo := documents.NewMemoryRepository()
	svc := documents.New(repo, &fakeIssuerPort{}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort())
	issuerID := uuid.New()

	seedDocument(t, repo, issuerID, "01", "accepted", time.Now())
	seedDocument(t, repo, issuerID, "01", "rejected", time.Now())
	seedDocument(t, repo, issuerID, "91", "accepted", time.Now())

	onlyInvoices, err := svc.ListDocuments(context.Background(), issuerID, documents.ListFilter{DianDocumentTypeCode: "01"})
	require.NoError(t, err)
	assert.Len(t, onlyInvoices, 2)

	onlyAccepted, err := svc.ListDocuments(context.Background(), issuerID, documents.ListFilter{Status: documents.StatusAccepted})
	require.NoError(t, err)
	assert.Len(t, onlyAccepted, 2)

	both, err := svc.ListDocuments(context.Background(), issuerID, documents.ListFilter{DianDocumentTypeCode: "01", Status: documents.StatusAccepted})
	require.NoError(t, err)
	assert.Len(t, both, 1)
}

func TestListDocuments_FiltersByDateRange(t *testing.T) {
	repo := documents.NewMemoryRepository()
	svc := documents.New(repo, &fakeIssuerPort{}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort())
	issuerID := uuid.New()

	seedDocument(t, repo, issuerID, "01", "accepted", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	seedDocument(t, repo, issuerID, "01", "accepted", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	seedDocument(t, repo, issuerID, "01", "accepted", time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))

	got, err := svc.ListDocuments(context.Background(), issuerID, documents.ListFilter{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, 2025, got[0].IssueDate.Year())
}

func TestListDocuments_LimitNormalization(t *testing.T) {
	repo := documents.NewMemoryRepository()
	svc := documents.New(repo, &fakeIssuerPort{}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort())
	issuerID := uuid.New()

	for i := 0; i < 5; i++ {
		seedDocument(t, repo, issuerID, "01", "accepted", time.Now())
	}

	// Limit <= 0 toma el default, nunca "sin límite".
	got, err := svc.ListDocuments(context.Background(), issuerID, documents.ListFilter{Limit: 0})
	require.NoError(t, err)
	assert.Len(t, got, 5)

	limited, err := svc.ListDocuments(context.Background(), issuerID, documents.ListFilter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, limited, 2)

	overMax, err := svc.ListDocuments(context.Background(), issuerID, documents.ListFilter{Limit: documents.MaxListLimit + 1000})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(overMax), documents.MaxListLimit)
}
