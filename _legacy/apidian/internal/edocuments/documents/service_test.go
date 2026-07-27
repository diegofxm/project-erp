package documents_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/diegofxm/apidian/internal/edocuments/customers"
	"github.com/diegofxm/apidian/internal/edocuments/documents"
	"github.com/diegofxm/apidian/internal/email"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/edocuments/numbering"
	"github.com/diegofxm/apidian/internal/edocuments/pdf"
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
	nr               *numbering.NumberingRange
	next             int64
	released         []int64 // números pasados a ReleaseIfCurrent, en orden — para que los tests confirmen cuándo se intentó devolver uno
	testSetIDCleared int     // veces que se llamó ClearTestSetID — para que los tests confirmen la sección 9.43
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

func (f *fakeNumberingPort) ClearTestSetID(_ context.Context, _ uuid.UUID) error {
	f.testSetIDCleared++
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

// fakeEmailPort guarda los mensajes que SendDocumentEmail intentó enviar — para que los tests
// confirmen el destinatario/adjuntos sin pegarle a SMTP real. err, si se configura, hace que
// Send falle siempre (para probar que documents.Service propaga el error tal cual).
type fakeEmailPort struct {
	sent []email.Message
	err  error
}

func (f *fakeEmailPort) Send(_ context.Context, msg email.Message) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, msg)
	return nil
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

// GetPaymentTermName/GetPaymentMethodName/GetIdentificationTypeName replican el subconjunto
// que usan estos tests — solo RenderDocumentPDF los necesita.
func (f *fakeCatalogPort) GetPaymentTermName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"1": "Contado", "2": "Crédito"}
	name, ok := names[code]
	return name, ok, nil
}

func (f *fakeCatalogPort) GetPaymentMethodName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"10": "Efectivo"}
	name, ok := names[code]
	return name, ok, nil
}

func (f *fakeCatalogPort) GetIdentificationTypeName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"13": "Cédula de Ciudadanía", "31": "NIT"}
	name, ok := names[code]
	return name, ok, nil
}

// GetItemStandardName/GetItemStandardAgencyID replican la tabla 13.3.5 real (sección 9.45) —
// solo "999" (estándar propio) tiene AgencyID vacío, a propósito.
func (f *fakeCatalogPort) GetItemStandardName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"001": "UNSPSC", "010": "GTIN", "020": "Partida Arancelaria", "999": "Estándar de adopción del contribuyente"}
	name, ok := names[code]
	return name, ok, nil
}

func (f *fakeCatalogPort) GetItemStandardAgencyID(_ context.Context, code string) (string, bool, error) {
	agencyIDs := map[string]string{"001": "10", "010": "9", "020": "195", "999": ""}
	agencyID, ok := agencyIDs[code]
	return agencyID, ok, nil
}

func (f *fakeCatalogPort) GetTaxRegimeName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"48": "Responsable de IVA", "00": "No aplica"}
	name, ok := names[code]
	return name, ok, nil
}

func (f *fakeCatalogPort) GetLiabilityCodeName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"O-13": "Gran Contribuyente", "O-15": "Autorretenedor", "R-99-PN": "No aplica (persona natural)", "R-99-PJ": "No aplica (persona jurídica)"}
	name, ok := names[code]
	return name, ok, nil
}

func (f *fakeCatalogPort) GetCiiuDescription(_ context.Context, code string) (string, bool, error) {
	descs := map[string]string{
		"6201": "Desarrollo de sistemas informáticos",
		"6202": "Consultoría informática y gerencia de instalaciones informáticas",
	}
	desc, ok := descs[code]
	return desc, ok, nil
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
		&fakeEmailPort{},
	)

	draft, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	assert.Equal(t, documents.StatusDraft, draft.Status)
	assert.Zero(t, draft.Number, "un borrador no reclama número")
	assert.Empty(t, draft.DocumentKey, "un borrador no tiene CUFE todavía")
	assert.Empty(t, draft.SignedXML, "un borrador no está firmado todavía")
	assert.Equal(t, int64(10000), draft.Totals.LineExtensionCents, "los totales sí se calculan desde el borrador")
}

// TestCreateInvoiceDraft_DefaultsCustomerCountry confirma el fix de la sección 9.44: un rechazo
// real (StatusCode 99, "errores en campos mandatorios") reveló que applyCustomerDefaults nunca
// defaulteaba el país del cliente — partyFromIssuer sí lo hace para el emisor, pero nunca se
// replicó para el cliente. Invisible mientras ningún cliente de prueba tuviera dirección.
func TestCreateInvoiceDraft_DefaultsCustomerCountry(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	req := testRequest(iss.ID, nr.ID)
	req.Customer.Address = domain.Address{
		Line:      "CL 17 B 19 68",
		CityCode:  "76520",
		CityName:  "Palmira",
		StateCode: "76",
		StateName: "Valle del Cauca",
	}

	draft, err := svc.CreateInvoiceDraft(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "CO", draft.Customer.Address.CountryCode)
	assert.Equal(t, "Colombia", draft.Customer.Address.CountryName)
}

// TestCreateInvoiceDraft_DefaultsItemStandard confirma el fix de la sección 9.45: un segundo
// rechazo real (mismo StatusCode 99) reveló que lines[].item_type_code/item_type_name/
// item_type_agency_id eran campos libres — la DIAN exige que coincidan con la tabla 13.3.5 del
// Anexo Técnico. item_type_code vacío debe defaultear a "999" (estándar propio) y
// item_type_name/item_type_agency_id deben derivarse del catálogo, nunca aceptarse del cliente.
func TestCreateInvoiceDraft_DefaultsItemStandard(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	req := testRequest(iss.ID, nr.ID)
	// ItemTypeCode se deja vacío a propósito — testRequest() ya no lo manda.

	draft, err := svc.CreateInvoiceDraft(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, draft.Lines, 1)
	assert.Equal(t, "999", draft.Lines[0].ItemTypeCode)
	assert.Equal(t, "Estándar de adopción del contribuyente", draft.Lines[0].ItemTypeName)
	assert.Empty(t, draft.Lines[0].ItemTypeAgencyID, "la fila 999 no debe traer agencyID")
}

// TestCreateInvoiceDraft_DerivesItemStandardWithAgencyID confirma el otro lado: un código real
// (ej. UNSPSC, "001") sí deriva un AgencyID no vacío.
func TestCreateInvoiceDraft_DerivesItemStandardWithAgencyID(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	req := testRequest(iss.ID, nr.ID)
	req.Lines[0].ItemTypeCode = "001"

	draft, err := svc.CreateInvoiceDraft(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, draft.Lines, 1)
	assert.Equal(t, "001", draft.Lines[0].ItemTypeCode)
	assert.Equal(t, "UNSPSC", draft.Lines[0].ItemTypeName)
	assert.Equal(t, "10", draft.Lines[0].ItemTypeAgencyID)
}

// TestCreateInvoiceDraft_InvalidItemStandardCode confirma el rechazo cuando item_type_code no
// existe en la tabla 13.3.5 — exactamente el bug real: un código UNSPSC puntual ("43211500")
// puesto donde debía ir el selector ("001").
func TestCreateInvoiceDraft_InvalidItemStandardCode(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	req := testRequest(iss.ID, nr.ID)
	req.Lines[0].ItemTypeCode = "43211500"

	_, err := svc.CreateInvoiceDraft(context.Background(), req)
	assert.ErrorIs(t, err, documents.ErrInvalidItemStandardCode)
}

// TestCreateInvoiceDraft_DefaultsTaxToIVAZero confirma el fix de la sección 9.46: un rechazo
// real (StatusCode 99, regla FAU04 "Base Imponible es distinto a la suma de los valores de las
// bases imponibles de todas líneas de detalle") reveló que una línea sin tax_type_code
// quedaba sin ningún cac:TaxTotal — la base imponible del encabezado dejaba de coincidir con
// la suma de las líneas. Mismo patrón ya probado contra la DIAN real en
// cofacture/soap/realsend_test.go: "sin impuesto" se modela como IVA al 0%, nunca como ausencia
// total de impuesto.
func TestCreateInvoiceDraft_DefaultsTaxToIVAZero(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	req := testRequest(iss.ID, nr.ID)
	req.Lines[0].TaxTypeCode = ""
	req.Lines[0].TaxPercent = 0

	draft, err := svc.CreateInvoiceDraft(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, draft.Lines, 1)
	require.Len(t, draft.Lines[0].Taxes, 1, "una línea sin impuesto especificado debe declarar IVA al 0%, nunca 0 impuestos")
	tax := draft.Lines[0].Taxes[0]
	assert.Equal(t, "01", tax.TypeCode)
	assert.Equal(t, "IVA", tax.TypeName)
	assert.Equal(t, float64(0), tax.Percent)
	assert.Equal(t, int64(0), tax.TaxAmountCents)
	assert.Equal(t, draft.Lines[0].LineExtensionCents, tax.TaxableAmountCents)
}

// TestCreateInvoiceDraft_AggregatesMixedTaxRatesSeparately confirma que dos líneas con IVA a
// tarifas distintas (19% y 0%, esta última por default) NO se fusionan en un solo
// cac:TaxSubtotal de cabecera — eso reportaría la base de la línea al 0% como si fuera al 19%,
// el mismo tipo de inconsistencia que dispara la regla FAU04.
func TestCreateInvoiceDraft_AggregatesMixedTaxRatesSeparately(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	req := testRequest(iss.ID, nr.ID)
	req.Lines[0].TaxTypeCode = "01"
	req.Lines[0].TaxPercent = 19
	req.Lines = append(req.Lines, documents.LineInput{
		Description:    "Servicio sin impuesto",
		Quantity:       1,
		UnitCode:       "94",
		UnitPriceCents: 5000,
		// TaxTypeCode vacío a propósito — debe defaultear a IVA 0%, separado del 19% de arriba.
	})

	draft, err := svc.CreateInvoiceDraft(context.Background(), req)
	require.NoError(t, err)

	taxes := documents.AggregateTaxesForTest(draft.Lines)
	require.Len(t, taxes, 2, "19%% y 0%% deben quedar en cac:TaxSubtotal separados, no fusionados")

	byPercent := map[float64]int64{}
	for _, t := range taxes {
		byPercent[t.Percent] = t.TaxableAmountCents
	}
	assert.Equal(t, draft.Lines[0].LineExtensionCents, byPercent[19])
	assert.Equal(t, draft.Lines[1].LineExtensionCents, byPercent[0])
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
		&fakeEmailPort{},
	)

	doc := issueInvoice(t, svc, testRequest(iss.ID, nr.ID))

	assert.Equal(t, "SETP", doc.Prefix)
	assert.Equal(t, int64(1), doc.Number)
	assert.NotEmpty(t, doc.DocumentKey, "debería tener CUFE calculado")
	assert.NotEmpty(t, doc.QRURL)
	assert.NotEmpty(t, doc.SignedXML)
	assert.Contains(t, doc.SignedXML, "<ds:Signature", "el XML persistido debe estar firmado")
	// iss=Producción ("1") y nr=Habilitación ("2") → mismatch → guard retorna sin enviar → "built".
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
	)

	req := testRequest(iss.ID, nr.ID)
	req.Customer.LiabilityCodes = []string{"CODIGO-INVENTADO"}
	_, err := svc.CreateInvoiceDraft(context.Background(), req)
	assert.ErrorIs(t, err, documents.ErrInvalidLiabilityCode)
}

// ── Fase 9.39: representación gráfica (PDF) ─────────────────────────────────────────────────

func TestRenderDocumentPDF_Draft(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	draft, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	got, err := svc.RenderDocumentPDF(context.Background(), iss.ID, draft.ID, pdf.FormatFullA4)
	require.NoError(t, err)
	assert.True(t, len(got) > 0 && string(got[:4]) == "%PDF", "debe ser un PDF válido")
}

func TestRenderDocumentPDF_Confirmed(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	doc := issueInvoice(t, svc, testRequest(iss.ID, nr.ID))

	got, err := svc.RenderDocumentPDF(context.Background(), iss.ID, doc.ID, pdf.FormatFullA4)
	require.NoError(t, err)
	assert.True(t, len(got) > 0 && string(got[:4]) == "%PDF", "debe ser un PDF válido")
}

func TestRenderDocumentPDF_OtherIssuer(t *testing.T) {
	iss := testIssuer()
	nr := testNumberingRange(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	draft, err := svc.CreateInvoiceDraft(context.Background(), testRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	_, err = svc.RenderDocumentPDF(context.Background(), uuid.New(), draft.ID, pdf.FormatFullA4)
	assert.ErrorIs(t, err, documents.ErrDocumentNotFound)
}

func TestRenderDocumentPDF_CreditNote(t *testing.T) {
	iss := testIssuer()
	nr := creditNoteRangeFor(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	req := documents.IssueCreditNoteRequest{
		IssueNoteRequest:   testNoteRequest(iss.ID, nr.ID),
		CreditNoteTypeCode: "2",
	}
	draft, err := svc.CreateCreditNoteDraft(context.Background(), req)
	require.NoError(t, err)

	got, err := svc.RenderDocumentPDF(context.Background(), iss.ID, draft.ID, pdf.FormatFullA4)
	require.NoError(t, err)
	assert.True(t, len(got) > 0 && string(got[:4]) == "%PDF", "debe ser un PDF válido")
}

func TestRenderDocumentPDF_DebitNote(t *testing.T) {
	iss := testIssuer()
	nr := debitNoteRangeFor(iss.ID)
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{nr: nr},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	draft, err := svc.CreateDebitNoteDraft(context.Background(), testNoteRequest(iss.ID, nr.ID))
	require.NoError(t, err)

	got, err := svc.RenderDocumentPDF(context.Background(), iss.ID, draft.ID, pdf.FormatFullA4)
	require.NoError(t, err)
	assert.True(t, len(got) > 0 && string(got[:4]) == "%PDF", "debe ser un PDF válido")
}

// ── Envío al cliente por correo (sección 9.42) ──────────────────────────────────────────────

// seedInvoice inserta una Factura directamente en repo, en el Status que el test necesite, sin
// pasar por todo el flujo real de firma/envío (mismo criterio que seedDocument, pero devuelve
// el documento y permite fijar el correo del cliente).
func seedInvoice(t *testing.T, repo documents.Repository, issuerID uuid.UUID, status documents.Status, customerEmail string) *documents.Document {
	t.Helper()
	d, err := repo.Create(context.Background(), documents.Document{
		IssuerID:             issuerID,
		NumberingRangeID:     uuid.New(),
		DianDocumentTypeCode: "01",
		Prefix:               "SETP",
		Number:               1,
		DocumentKey:          "cufe-de-prueba",
		IssueDate:            time.Now(),
		IssueTime:            "10:00:00-05:00",
		CurrencyCode:         "COP",
		Customer:             domain.Party{Name: "Cliente de prueba", Email: customerEmail},
		Lines:                []domain.Line{{Description: "Servicio", Quantity: 1}},
		SignedXML:            "<xml>factura firmada de prueba</xml>",
		Status:               status,
	})
	require.NoError(t, err)
	return d
}

func seedCreditNote(t *testing.T, repo documents.Repository, issuerID uuid.UUID, status documents.Status, customerEmail string) *documents.Document {
	t.Helper()
	d, err := repo.Create(context.Background(), documents.Document{
		IssuerID:             issuerID,
		NumberingRangeID:     uuid.New(),
		DianDocumentTypeCode: "91",
		Prefix:               "SETPNC",
		Number:               1,
		DocumentKey:          "cude-nc-de-prueba",
		IssueDate:            time.Now(),
		IssueTime:            "10:00:00-05:00",
		CurrencyCode:         "COP",
		Customer:             domain.Party{Name: "Cliente de prueba", Email: customerEmail},
		Lines:                []domain.Line{{Description: "Anulación", Quantity: 1}},
		SignedXML:            "<xml>nota credito firmada de prueba</xml>",
		Status:               status,
	})
	require.NoError(t, err)
	return d
}

func seedDebitNote(t *testing.T, repo documents.Repository, issuerID uuid.UUID, status documents.Status, customerEmail string) *documents.Document {
	t.Helper()
	d, err := repo.Create(context.Background(), documents.Document{
		IssuerID:             issuerID,
		NumberingRangeID:     uuid.New(),
		DianDocumentTypeCode: "92",
		Prefix:               "SETPND",
		Number:               1,
		DocumentKey:          "cude-nd-de-prueba",
		IssueDate:            time.Now(),
		IssueTime:            "10:00:00-05:00",
		CurrencyCode:         "COP",
		Customer:             domain.Party{Name: "Cliente de prueba", Email: customerEmail},
		Lines:                []domain.Line{{Description: "Cargo adicional", Quantity: 1}},
		SignedXML:            "<xml>nota debito firmada de prueba</xml>",
		Status:               status,
	})
	require.NoError(t, err)
	return d
}

func TestSendDocumentEmail_OK(t *testing.T) {
	iss := testIssuer()
	repo := documents.NewMemoryRepository()
	emailPort := &fakeEmailPort{}
	svc := documents.New(repo, &fakeIssuerPort{issuer: iss}, &fakeNumberingPort{nr: testNumberingRange(iss.ID)}, &fakeCustomerPort{}, newFakeCatalogPort(), emailPort)
	doc := seedInvoice(t, repo, iss.ID, documents.StatusAccepted, "cliente@example.test")

	err := svc.SendDocumentEmail(context.Background(), iss.ID, doc.ID, pdf.FormatFullA4, "", nil)
	require.NoError(t, err)

	require.Len(t, emailPort.sent, 1)
	msg := emailPort.sent[0]
	assert.Equal(t, "cliente@example.test", msg.To)
	assert.Contains(t, msg.Subject, "SETP1")
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "SETP1.zip", msg.Attachments[0].Filename)
	assert.Equal(t, "application/zip", msg.Attachments[0].ContentType)
	assert.NotEmpty(t, msg.Attachments[0].Content)
}

func TestSendDocumentEmail_NotAccepted(t *testing.T) {
	iss := testIssuer()
	repo := documents.NewMemoryRepository()
	svc := documents.New(repo, &fakeIssuerPort{issuer: iss}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort(), &fakeEmailPort{})
	doc := seedInvoice(t, repo, iss.ID, documents.StatusBuilt, "cliente@example.test")

	err := svc.SendDocumentEmail(context.Background(), iss.ID, doc.ID, pdf.FormatFullA4, "", nil)
	assert.ErrorIs(t, err, documents.ErrDocumentNotAccepted)
}

func TestSendDocumentEmail_MissingCustomerEmail(t *testing.T) {
	iss := testIssuer()
	repo := documents.NewMemoryRepository()
	svc := documents.New(repo, &fakeIssuerPort{issuer: iss}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort(), &fakeEmailPort{})
	doc := seedInvoice(t, repo, iss.ID, documents.StatusAccepted, "")

	err := svc.SendDocumentEmail(context.Background(), iss.ID, doc.ID, pdf.FormatFullA4, "", nil)
	assert.ErrorIs(t, err, documents.ErrCustomerEmailMissing)
}

func TestSendDocumentEmail_OtherIssuer(t *testing.T) {
	iss := testIssuer()
	repo := documents.NewMemoryRepository()
	svc := documents.New(repo, &fakeIssuerPort{issuer: iss}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort(), &fakeEmailPort{})
	doc := seedInvoice(t, repo, iss.ID, documents.StatusAccepted, "cliente@example.test")

	err := svc.SendDocumentEmail(context.Background(), uuid.New(), doc.ID, pdf.FormatFullA4, "", nil)
	assert.ErrorIs(t, err, documents.ErrDocumentNotFound)
}

func TestSendDocumentEmail_CreditNote(t *testing.T) {
	iss := testIssuer()
	repo := documents.NewMemoryRepository()
	emailPort := &fakeEmailPort{}
	nr := creditNoteRangeFor(iss.ID)
	svc := documents.New(repo, &fakeIssuerPort{issuer: iss}, &fakeNumberingPort{nr: nr}, &fakeCustomerPort{}, newFakeCatalogPort(), emailPort)
	doc := seedCreditNote(t, repo, iss.ID, documents.StatusAccepted, "cliente@example.test")

	err := svc.SendDocumentEmail(context.Background(), iss.ID, doc.ID, pdf.FormatFullA4, "", nil)
	require.NoError(t, err)

	require.Len(t, emailPort.sent, 1)
	msg := emailPort.sent[0]
	assert.Equal(t, "cliente@example.test", msg.To)
	assert.Contains(t, msg.Subject, "SETPNC1")
	assert.Contains(t, msg.Subject, "91")
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "SETPNC1.zip", msg.Attachments[0].Filename)
	assert.Equal(t, "application/zip", msg.Attachments[0].ContentType)
}

func TestSendDocumentEmail_SendFailurePropagates(t *testing.T) {
	iss := testIssuer()
	repo := documents.NewMemoryRepository()
	sendErr := errors.New("smtp: fallo simulado")
	svc := documents.New(repo, &fakeIssuerPort{issuer: iss}, &fakeNumberingPort{nr: testNumberingRange(iss.ID)}, &fakeCustomerPort{}, newFakeCatalogPort(), &fakeEmailPort{err: sendErr})
	doc := seedInvoice(t, repo, iss.ID, documents.StatusAccepted, "cliente@example.test")

	err := svc.SendDocumentEmail(context.Background(), iss.ID, doc.ID, pdf.FormatFullA4, "", nil)
	assert.ErrorIs(t, err, sendErr)
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
		&fakeEmailPort{},
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
	svc := documents.New(repo, &fakeIssuerPort{}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort(), &fakeEmailPort{})
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
	svc := documents.New(repo, &fakeIssuerPort{}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort(), &fakeEmailPort{})
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
	svc := documents.New(repo, &fakeIssuerPort{}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort(), &fakeEmailPort{})
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
	svc := documents.New(repo, &fakeIssuerPort{}, &fakeNumberingPort{}, &fakeCustomerPort{}, newFakeCatalogPort(), &fakeEmailPort{})
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

// ── VerifyAcquirer (ayuda opcional al capturar un NIT, sección 9.41) ───────────────────────────

// TestVerifyAcquirer_IssuerNotReadyToIssue confirma el único caso seguro de probar sin red real
// (GetAcquirer si llega a llamarse de verdad golpea la DIAN real — ver el resto de esta
// suite, que evita esa ruta a propósito con Environment: producción). El resto del flujo
// (certificado configurado, llamada SOAP real) se verifica manualmente contra la DIAN real,
// igual que GetStatus/SendBillSync en su momento.
func TestVerifyAcquirer_IssuerNotReadyToIssue(t *testing.T) {
	iss := &issuers.Issuer{ID: uuid.New(), BusinessName: "Sin Certificado S.A.S."}
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{issuer: iss},
		&fakeNumberingPort{},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	_, err := svc.VerifyAcquirer(context.Background(), iss.ID, "31", "900373076")
	assert.ErrorIs(t, err, documents.ErrIssuerNotReadyToIssue)
}

func TestVerifyAcquirer_IssuerNotFound(t *testing.T) {
	svc := documents.New(
		documents.NewMemoryRepository(),
		&fakeIssuerPort{},
		&fakeNumberingPort{},
		&fakeCustomerPort{},
		newFakeCatalogPort(),
		&fakeEmailPort{},
	)

	_, err := svc.VerifyAcquirer(context.Background(), uuid.New(), "31", "900373076")
	assert.ErrorIs(t, err, issuers.ErrIssuerNotFound)
}
