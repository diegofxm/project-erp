package application_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/erp/internal/electronic/application"
	"github.com/diegofxm/erp/internal/electronic/domain"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeDocumentRepo struct {
	doc           domain.Document
	confirmCalled bool
	statusUpdates []domain.Status
}

func (f *fakeDocumentRepo) Create(context.Context, domain.Document) (*domain.Document, error) {
	return nil, nil
}
func (f *fakeDocumentRepo) GetByID(context.Context, uuid.UUID) (*domain.Document, error) {
	d := f.doc
	return &d, nil
}
func (f *fakeDocumentRepo) GetByDocumentKey(context.Context, uuid.UUID, string) (*domain.Document, error) {
	return nil, nil
}
func (f *fakeDocumentRepo) UpdateDraft(context.Context, domain.Document) (*domain.Document, error) {
	return nil, nil
}
func (f *fakeDocumentRepo) Confirm(_ context.Context, d domain.Document) (*domain.Document, error) {
	f.confirmCalled = true
	f.doc = d
	return &d, nil
}
func (f *fakeDocumentRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (f *fakeDocumentRepo) UpdateDianStatus(_ context.Context, _ uuid.UUID, status domain.Status, trackID, statusCode, statusDescription, statusMessage, applicationResponseXML string) error {
	f.statusUpdates = append(f.statusUpdates, status)
	f.doc.Status = status
	f.doc.DianTrackID = trackID
	f.doc.DianStatusCode = statusCode
	f.doc.DianStatusDescription = statusDescription
	f.doc.DianStatusMessage = statusMessage
	f.doc.ApplicationResponseXML = applicationResponseXML
	return nil
}
func (f *fakeDocumentRepo) ListByCompany(context.Context, uuid.UUID, domain.ListFilter) ([]*domain.Document, error) {
	return nil, nil
}
func (f *fakeDocumentRepo) GetRelatedNotes(context.Context, uuid.UUID, string, int64) ([]domain.RelatedNote, error) {
	return nil, nil
}

type fakeNumberingRepo struct {
	nr               domain.NumberingRange
	nextNumber       int64
	releasedNumbers  []int64
	clearedTestSetID bool
}

func (f *fakeNumberingRepo) Create(context.Context, domain.NumberingRange) (*domain.NumberingRange, error) {
	return nil, nil
}
func (f *fakeNumberingRepo) GetByID(context.Context, uuid.UUID) (*domain.NumberingRange, error) {
	nr := f.nr
	return &nr, nil
}
func (f *fakeNumberingRepo) ClaimNext(context.Context, uuid.UUID) (int64, error) {
	f.nextNumber++
	return f.nextNumber, nil
}
func (f *fakeNumberingRepo) ReleaseIfCurrent(_ context.Context, _ uuid.UUID, number int64) error {
	f.releasedNumbers = append(f.releasedNumbers, number)
	return nil
}
func (f *fakeNumberingRepo) ClearTestSetID(context.Context, uuid.UUID) error {
	f.clearedTestSetID = true
	return nil
}
func (f *fakeNumberingRepo) Deactivate(context.Context, uuid.UUID) error { return nil }
func (f *fakeNumberingRepo) Activate(context.Context, uuid.UUID) error  { return nil }
func (f *fakeNumberingRepo) ListByCompany(context.Context, uuid.UUID, string) ([]*domain.NumberingRange, error) {
	return nil, nil
}

type fakeCompanyPort struct {
	company domain.CompanyInfo
}

func (f *fakeCompanyPort) GetCompany(context.Context, uuid.UUID) (*domain.CompanyInfo, error) {
	c := f.company
	return &c, nil
}

type fakeBuilder struct{}

func (fakeBuilder) SignedInvoiceXML(cofdom.Invoice, []byte, string, time.Time) ([]byte, error) {
	return []byte("<xml/>"), nil
}
func (fakeBuilder) SignedCreditNoteXML(cofdom.CreditNote, []byte, string, time.Time) ([]byte, error) {
	return []byte("<xml/>"), nil
}
func (fakeBuilder) SignedDebitNoteXML(cofdom.DebitNote, []byte, string, time.Time) ([]byte, error) {
	return []byte("<xml/>"), nil
}
func (fakeBuilder) SignedSupportDocumentXML(cofdom.Invoice, []byte, string, time.Time) ([]byte, error) {
	return []byte("<xml/>"), nil
}
func (fakeBuilder) SignedAdjustmentNoteXML(cofdom.AdjustmentNote, []byte, string, time.Time) ([]byte, error) {
	return []byte("<xml/>"), nil
}

type fakeZipper struct{}

func (fakeZipper) Zip(_ string, _ string, _ int, _ int64, _ []byte) (string, []byte, error) {
	return "doc.zip", []byte("zipbytes"), nil
}

// fakeSender permite fijar el comportamiento de cada operación por separado y cuenta cuántas
// veces se invocó cada una, para verificar la decisión síncrona/asíncrona sin inspeccionar
// estado interno del caso de uso.
type fakeSender struct {
	sendBillSyncErr    error
	sendBillSyncResult *domain.SendResult

	sendBillAsyncErr error
	sendBillAsyncKey string

	sendTestSetErr error
	sendTestSetKey string

	// pollResults se consumen en orden, uno por llamada a PollStatusZip.
	pollResults []*domain.SendResult

	syncCalls    int
	asyncCalls   int
	testSetCalls int
	pollCalls    int
}

func (f *fakeSender) SendBillSync(string, []byte, []byte, string, string) (*domain.SendResult, error) {
	f.syncCalls++
	if f.sendBillSyncErr != nil {
		return nil, f.sendBillSyncErr
	}
	if f.sendBillSyncResult != nil {
		return f.sendBillSyncResult, nil
	}
	return &domain.SendResult{IsValid: true}, nil
}
func (f *fakeSender) SendBillAsync(string, []byte, []byte, string, string) (string, error) {
	f.asyncCalls++
	if f.sendBillAsyncErr != nil {
		return "", f.sendBillAsyncErr
	}
	return f.sendBillAsyncKey, nil
}
func (f *fakeSender) SendTestSetAsync(string, []byte, string, []byte, string, string) (string, error) {
	f.testSetCalls++
	if f.sendTestSetErr != nil {
		return "", f.sendTestSetErr
	}
	return f.sendTestSetKey, nil
}
func (f *fakeSender) PollStatusZip(string, []byte, string, string) (*domain.SendResult, error) {
	idx := f.pollCalls
	f.pollCalls++
	if idx < len(f.pollResults) {
		return f.pollResults[idx], nil
	}
	return nil, nil // sondeo agotado, sin respuesta útil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newConfirmTestUseCase(t *testing.T) (uc *application.ConfirmUseCase, docs *fakeDocumentRepo, numbering *fakeNumberingRepo, sender *fakeSender) {
	t.Helper()
	// El sondeo agotado hace 5 sleeps de PollInterval -- se acorta para que esos casos de test
	// (StatusSent tras timeout) no tarden ~25s reales.
	orig := application.PollInterval
	application.PollInterval = time.Millisecond
	t.Cleanup(func() { application.PollInterval = orig })

	companyID := uuid.New()
	rangeID := uuid.New()
	docID := uuid.New()

	docs = &fakeDocumentRepo{doc: domain.Document{
		ID:                   docID,
		CompanyID:            companyID,
		NumberingRangeID:     rangeID,
		DianDocumentTypeCode: "01", // FE
		Status:               domain.StatusDraft,
		CurrencyCode:         "COP",
		Customer:             cofdom.Party{Name: "Cliente Test"},
		Lines: []cofdom.Line{
			{Description: "Item", Quantity: 1, UnitPriceCents: 100000},
		},
		PaymentMeans: []cofdom.PaymentMean{{Code: "1"}},
	}}
	numbering = &fakeNumberingRepo{nr: domain.NumberingRange{
		ID:                   rangeID,
		CompanyID:            companyID,
		DianDocumentTypeCode: "01",
		Prefix:               "SETP",
		RangeFrom:            1,
		Environment:          domain.EnvHabilitacion,
		IsActive:             true,
	}}
	companies := &fakeCompanyPort{company: domain.CompanyInfo{
		ID:           companyID,
		NIT:          "900123456",
		BusinessName: "Empresa Test",
		Environment:  domain.EnvHabilitacion,
		SoftwareID:   "sw-id",
		SoftwarePIN:  "1234",
		Certificate:  []byte("cert-bytes"),
	}}
	sender = &fakeSender{}

	uc = application.NewConfirmUseCase(docs, numbering, companies, fakeBuilder{}, fakeZipper{}, sender)
	return uc, docs, numbering, sender
}

// ── validación de entrada ────────────────────────────────────────────────────

func TestConfirm_WrongCompany_ReturnsDocumentNotFound(t *testing.T) {
	uc, docs, _, _ := newConfirmTestUseCase(t)

	_, err := uc.Confirm(context.Background(), uuid.New(), docs.doc.ID)
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("esperaba ErrDocumentNotFound, got %v", err)
	}
}

func TestConfirm_NotDraft_ReturnsError(t *testing.T) {
	uc, docs, _, _ := newConfirmTestUseCase(t)
	docs.doc.Status = domain.StatusAccepted

	_, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if !errors.Is(err, domain.ErrDocumentNotDraft) {
		t.Fatalf("esperaba ErrDocumentNotDraft, got %v", err)
	}
}

func TestConfirm_CreditNote_MissingBillingReference_ReturnsError(t *testing.T) {
	uc, docs, _, _ := newConfirmTestUseCase(t)
	docs.doc.DianDocumentTypeCode = "91" // NC

	_, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if !errors.Is(err, domain.ErrMissingBillingReference) {
		t.Fatalf("esperaba ErrMissingBillingReference, got %v", err)
	}
}

func TestConfirm_SupportDocument_MissingSupplier_ReturnsError(t *testing.T) {
	uc, docs, _, _ := newConfirmTestUseCase(t)
	docs.doc.DianDocumentTypeCode = "05" // DS

	_, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if !errors.Is(err, domain.ErrMissingSupplier) {
		t.Fatalf("esperaba ErrMissingSupplier, got %v", err)
	}
}

// ── decisión síncrona/asíncrona ──────────────────────────────────────────────

func TestConfirm_Production_UsesSyncSend_SameEnvironment(t *testing.T) {
	_, _, numbering, _ := newConfirmTestUseCase(t)
	numbering.nr.Environment = domain.EnvProduccion
	numbering.nr.TestSetID = "ws-123" // presente, pero producción ignora TestSetID

	// company y rango deben coincidir en ambiente para no disparar EnvironmentMismatch --
	// aquí se prueba la decisión sync/async, no esa validación.
	uc, docs, _, sender := rebuildWithCompanyEnv(t, domain.EnvProduccion, numbering)

	_, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if sender.syncCalls != 1 {
		t.Fatalf("esperaba 1 llamada a SendBillSync, got %d", sender.syncCalls)
	}
	if sender.testSetCalls != 0 {
		t.Fatalf("producción no debería usar SendTestSetAsync, got %d llamadas", sender.testSetCalls)
	}
}

func TestConfirm_Habilitacion_NoTestSetID_UsesSyncSend(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = "" // habilitación sin TestSetID también va por la vía síncrona

	_, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if sender.syncCalls != 1 || sender.testSetCalls != 0 {
		t.Fatalf("esperaba solo SendBillSync, got sync=%d testset=%d", sender.syncCalls, sender.testSetCalls)
	}
}

func TestConfirm_Habilitacion_WithTestSetID_UsesTestSetAsyncFlow(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = "ws-123"
	sender.sendTestSetKey = "zip-key-1"
	sender.pollResults = []*domain.SendResult{
		{IsValid: true, StatusCode: "1", IsTestSetClosed: false},
	}

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if sender.testSetCalls != 1 || sender.syncCalls != 0 {
		t.Fatalf("esperaba solo SendTestSetAsync, got testset=%d sync=%d", sender.testSetCalls, sender.syncCalls)
	}
	if doc.Status != domain.StatusAccepted {
		t.Fatalf("esperaba StatusAccepted, got %s", doc.Status)
	}
}

func TestConfirm_TestSetClosed_FallsBackToSyncAndClearsTestSetID(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = "ws-123"
	sender.sendTestSetKey = "zip-key-1"
	sender.pollResults = []*domain.SendResult{
		{IsTestSetClosed: true}, // la DIAN avisa que el set de pruebas ya se cerró
	}

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !numbering.clearedTestSetID {
		t.Fatal("esperaba que se limpiara el TestSetID del rango")
	}
	if sender.testSetCalls != 1 || sender.syncCalls != 1 {
		t.Fatalf("esperaba testset=1 (probado) y sync=1 (fallback), got testset=%d sync=%d", sender.testSetCalls, sender.syncCalls)
	}
	if doc.Status != domain.StatusAccepted {
		t.Fatalf("esperaba StatusAccepted vía el fallback síncrono, got %s", doc.Status)
	}
}

// ── environment mismatch (punto 05) ──────────────────────────────────────────

func TestConfirm_EnvironmentMismatch_ReleasesNumberWithoutSending(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.Environment = domain.EnvProduccion // company sigue en Habilitacion (fixture)

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error de Go, el mismatch se refleja en el estado: %v", err)
	}
	if doc.Status != domain.StatusEnvironmentMismatch {
		t.Fatalf("esperaba StatusEnvironmentMismatch, got %s", doc.Status)
	}
	if len(numbering.releasedNumbers) != 1 || numbering.releasedNumbers[0] != 1 {
		t.Fatalf("esperaba liberar el consecutivo 1, got %v", numbering.releasedNumbers)
	}
	if sender.syncCalls+sender.asyncCalls+sender.testSetCalls != 0 {
		t.Fatal("no debería haberse intentado ningún envío a la DIAN")
	}
}

// ── reconciliación de errores ambiguos vs. explícitos (puntos 02/03) ─────────

func TestConfirm_DianRejectedSync_MarksSendErrorAndReleasesNumber(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = ""
	sender.sendBillSyncErr = fmt.Errorf("%w: nit inválido", domain.ErrDianRejectedSync)

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error de Go: %v", err)
	}
	if doc.Status != domain.StatusSendError {
		t.Fatalf("esperaba StatusSendError, got %s", doc.Status)
	}
	if len(numbering.releasedNumbers) != 1 {
		t.Fatalf("esperaba liberar el consecutivo (rechazo explícito, sin ambigüedad), got %v", numbering.releasedNumbers)
	}
	if sender.asyncCalls != 0 {
		t.Fatalf("un rechazo explícito no debería disparar el reintento asíncrono, got %d llamadas", sender.asyncCalls)
	}
}

func TestConfirm_TransportError_RetriesAsyncAndSucceeds(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = ""
	sender.sendBillSyncErr = errors.New("timeout de red") // ambiguo, no es un soap.Fault
	sender.sendBillAsyncKey = "zip-key-retry"
	sender.pollResults = []*domain.SendResult{
		{IsValid: true, StatusCode: "1"},
	}

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error de Go: %v", err)
	}
	if sender.asyncCalls != 1 {
		t.Fatalf("esperaba 1 reintento asíncrono tras el fallo síncrono ambiguo, got %d", sender.asyncCalls)
	}
	if doc.Status != domain.StatusAccepted {
		t.Fatalf("esperaba StatusAccepted (el reintento sí llegó), got %s", doc.Status)
	}
	if len(numbering.releasedNumbers) != 0 {
		t.Fatalf("un envío finalmente aceptado no debe liberar el consecutivo, got %v", numbering.releasedNumbers)
	}
}

func TestConfirm_TransportError_BothAmbiguous_MarksSendUnknown_DoesNotReleaseNumber(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = ""
	sender.sendBillSyncErr = errors.New("timeout de red")
	sender.sendBillAsyncErr = errors.New("conexión rechazada")

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error de Go: %v", err)
	}
	if doc.Status != domain.StatusSendUnknown {
		t.Fatalf("esperaba StatusSendUnknown, got %s", doc.Status)
	}
	if len(numbering.releasedNumbers) != 0 {
		t.Fatalf("StatusSendUnknown NO debe liberar el consecutivo (riesgo de doble facturación), got %v", numbering.releasedNumbers)
	}
}

func TestConfirm_TransportError_AsyncRetryRejectedByDian_MarksSendErrorAndReleases(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = ""
	sender.sendBillSyncErr = errors.New("timeout de red")
	sender.sendBillAsyncErr = fmt.Errorf("%w: xml malformado", domain.ErrDianRejectedSync)

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error de Go: %v", err)
	}
	if doc.Status != domain.StatusSendError {
		t.Fatalf("el reintento asíncrono rechazado explícitamente ya no es ambiguo, esperaba StatusSendError, got %s", doc.Status)
	}
	if len(numbering.releasedNumbers) != 1 {
		t.Fatalf("esperaba liberar el consecutivo, got %v", numbering.releasedNumbers)
	}
}

func TestConfirm_TransportError_AsyncRetryPollExhausted_LeavesStatusSentRecoverable(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = ""
	sender.sendBillSyncErr = errors.New("timeout de red")
	sender.sendBillAsyncKey = "zip-key-retry"
	sender.pollResults = nil // los 6 intentos de sondeo se agotan sin respuesta

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error de Go: %v", err)
	}
	if doc.Status != domain.StatusSent {
		t.Fatalf("esperaba StatusSent (recuperable vía CheckPendingStatus), got %s", doc.Status)
	}
	if doc.DianTrackID != "zip-key-retry" {
		t.Fatalf("esperaba conservar el zipKey para poder recuperar el documento después, got %q", doc.DianTrackID)
	}
	if len(numbering.releasedNumbers) != 0 {
		t.Fatalf("StatusSent no debe liberar el consecutivo, got %v", numbering.releasedNumbers)
	}
}

// ── resultado directo de la DIAN (no ambiguo) ────────────────────────────────

func TestConfirm_Rejected_ReleasesNumber(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = ""
	sender.sendBillSyncResult = &domain.SendResult{HasRejections: true, StatusMessage: "NIT no coincide"}

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error de Go: %v", err)
	}
	if doc.Status != domain.StatusRejected {
		t.Fatalf("esperaba StatusRejected, got %s", doc.Status)
	}
	if len(numbering.releasedNumbers) != 1 {
		t.Fatalf("esperaba liberar el consecutivo tras un rechazo, got %v", numbering.releasedNumbers)
	}
}

// TestConfirm_RespondedCUFEMismatch_WarnsNumberAlreadyConsumed cubre el diagnóstico real del
// 2026-08-11: cuando la DIAN responde con un CUFE (XmlDocumentKey) distinto al que este
// documento generó, no validó nuestro contenido -- devolvió el resultado de un envío anterior
// para el mismo consecutivo (típicamente quemado a mano en el portal de habilitación). El
// mensaje original de la DIAN describe ese documento ajeno y debe reemplazarse por una alerta
// clara en vez de mostrarse tal cual.
func TestConfirm_RespondedCUFEMismatch_WarnsNumberAlreadyConsumed(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = ""
	sender.sendBillSyncResult = &domain.SendResult{
		HasRejections:        true,
		StatusMessage:        "Nombre informado No corresponde al registrado en el RUT",
		RespondedDocumentKey: "cufe-de-un-documento-completamente-distinto",
	}

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error de Go: %v", err)
	}
	if doc.Status != domain.StatusRejected {
		t.Fatalf("esperaba StatusRejected, got %s", doc.Status)
	}
	if doc.DianStatusMessage == "Nombre informado No corresponde al registrado en el RUT" {
		t.Fatalf("el mensaje original de la DIAN (de un documento ajeno) no debió pasar tal cual, got %q", doc.DianStatusMessage)
	}
	if !strings.Contains(doc.DianStatusMessage, "ya fue usado anteriormente ante la DIAN") {
		t.Fatalf("esperaba la alerta de consecutivo ya consumido, got %q", doc.DianStatusMessage)
	}
}

// TestConfirm_RespondedCUFEMatch_NoWarning confirma que el caso normal (la DIAN responde sobre
// el mismo CUFE que enviamos) no dispara la alerta -- el StatusMessage original de la DIAN pasa
// intacto.
func TestConfirm_RespondedCUFEMatch_NoWarning(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = ""
	sender.sendBillSyncResult = &domain.SendResult{IsValid: true, StatusMessage: "ha sido autorizada"}

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error de Go: %v", err)
	}
	if doc.DianStatusMessage != "ha sido autorizada" {
		t.Fatalf("esperaba el mensaje original de la DIAN sin alterar, got %q", doc.DianStatusMessage)
	}
}

func TestConfirm_Accepted_DoesNotReleaseNumber(t *testing.T) {
	uc, docs, numbering, sender := newConfirmTestUseCase(t)
	numbering.nr.TestSetID = ""
	sender.sendBillSyncResult = &domain.SendResult{IsValid: true}

	doc, err := uc.Confirm(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error de Go: %v", err)
	}
	if doc.Status != domain.StatusAccepted {
		t.Fatalf("esperaba StatusAccepted, got %s", doc.Status)
	}
	if len(numbering.releasedNumbers) != 0 {
		t.Fatalf("un documento aceptado nunca debe liberar su consecutivo, got %v", numbering.releasedNumbers)
	}
}

// ── CheckPendingStatus (resolución manual de StatusSent) ─────────────────────

func TestCheckPendingStatus_NotSent_ReturnsError(t *testing.T) {
	uc, docs, _, _ := newConfirmTestUseCase(t)
	docs.doc.Status = domain.StatusAccepted

	_, err := uc.CheckPendingStatus(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if !errors.Is(err, domain.ErrDocumentNotPending) {
		t.Fatalf("esperaba ErrDocumentNotPending, got %v", err)
	}
}

func TestCheckPendingStatus_ResolvesFromNewPoll(t *testing.T) {
	uc, docs, _, sender := newConfirmTestUseCase(t)
	docs.doc.Status = domain.StatusSent
	docs.doc.DianTrackID = "zip-key-pending"
	sender.pollResults = []*domain.SendResult{
		{IsValid: true, StatusCode: "1"},
	}

	doc, err := uc.CheckPendingStatus(context.Background(), docs.doc.CompanyID, docs.doc.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if doc.Status != domain.StatusAccepted {
		t.Fatalf("esperaba que el nuevo sondeo resolviera a StatusAccepted, got %s", doc.Status)
	}
	if sender.pollCalls != 1 {
		t.Fatalf("esperaba 1 llamada a PollStatusZip, got %d", sender.pollCalls)
	}
}

// ── helper para el caso de producción, donde company.Environment debe coincidir con el rango ──

func rebuildWithCompanyEnv(t *testing.T, env domain.Environment, numbering *fakeNumberingRepo) (*application.ConfirmUseCase, *fakeDocumentRepo, *fakeNumberingRepo, *fakeSender) {
	t.Helper()
	companyID := numbering.nr.CompanyID
	docID := uuid.New()

	docs := &fakeDocumentRepo{doc: domain.Document{
		ID:                   docID,
		CompanyID:            companyID,
		NumberingRangeID:     numbering.nr.ID,
		DianDocumentTypeCode: "01",
		Status:               domain.StatusDraft,
		CurrencyCode:         "COP",
		Customer:             cofdom.Party{Name: "Cliente Test"},
		Lines: []cofdom.Line{
			{Description: "Item", Quantity: 1, UnitPriceCents: 100000},
		},
		PaymentMeans: []cofdom.PaymentMean{{Code: "1"}},
	}}
	companies := &fakeCompanyPort{company: domain.CompanyInfo{
		ID:           companyID,
		NIT:          "900123456",
		BusinessName: "Empresa Test",
		Environment:  env,
		SoftwareID:   "sw-id",
		SoftwarePIN:  "1234",
		Certificate:  []byte("cert-bytes"),
	}}
	sender := &fakeSender{}
	uc := application.NewConfirmUseCase(docs, numbering, companies, fakeBuilder{}, fakeZipper{}, sender)
	return uc, docs, numbering, sender
}
