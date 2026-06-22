package documents

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"strconv"
	"time"

	"github.com/beevik/etree"
	"github.com/diegofxm/api-dian/internal/issuers"
	"github.com/diegofxm/api-dian/internal/numbering"
	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/cude"
	"github.com/diegofxm/cofacture/cufe"
	"github.com/diegofxm/cofacture/dian"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
	"github.com/diegofxm/cofacture/zip"
	"github.com/google/uuid"
)

// Los valores fijos de abajo están confirmados contra documentos reales autorizados por la
// DIAN (cofacture/soap/realsend_test.go para Invoice/CreditNote) — no son arbitrarios.
// debitNoteProfileID es la única excepción: sigue el mismo patrón de nombre que las otras
// dos, pero no se ha confirmado contra un envío real todavía (ver Fase 9.9 del architecture
// doc — DebitNote no se ha enviado real, solo construido y firmado).
const (
	invoiceProfileID         = "DIAN 2.1: Factura Electrónica de Venta"
	invoiceOperationTypeCode = "10"
	invoiceHashType          = "CUFE-SHA384"
	invoiceDianDocumentType  = "01"

	creditNoteProfileID         = "DIAN 2.1: Nota Crédito de Factura Electrónica de Venta"
	creditNoteOperationTypeCode = "20"
	creditNoteHashType          = "CUDE-SHA384"
	creditNoteDianDocumentType  = "91"

	debitNoteProfileID         = "DIAN 2.1: Nota Débito de Factura Electrónica de Venta"
	debitNoteOperationTypeCode = "30"
	debitNoteHashType          = "CUDE-SHA384"
	debitNoteDianDocumentType  = "92"
)

// Service orquesta la emisión de documentos DIAN: reclama numeración, construye y firma el
// XML con cofacture, lo envía por SOAP, interpreta la respuesta, y persiste el resultado.
//
// Es el ÚNICO paquete de api-dian que importa cofacture directamente — ver
// docs/api-dian-architecture.md sección 4.1.
type Service struct {
	repo      Repository
	issuers   IssuerPort
	numbering NumberingPort
	customers CustomerPort
}

// New crea el servicio de documentos.
func New(repo Repository, issuerPort IssuerPort, numberingPort NumberingPort, customerPort CustomerPort) *Service {
	return &Service{repo: repo, issuers: issuerPort, numbering: numberingPort, customers: customerPort}
}

// IssueInvoiceRequest es el payload de emisión de una Factura Electrónica de Venta.
// Customer/Lines/PaymentMeans son pass-through: llegan tal cual y se persisten como
// snapshot junto con el documento (ver model.go).
type IssueInvoiceRequest struct {
	IssuerID         uuid.UUID
	NumberingRangeID uuid.UUID
	Customer         domain.Party
	Lines            []domain.Line
	PaymentMeans     []domain.PaymentMean
	Note             string
	CurrencyCode     string // default "COP" si vacío

	// CustomerID es opcional — referencia de solo trazabilidad a internal/customers (ver
	// model.go). nil si la petición no traía un cliente guardado.
	CustomerID *uuid.UUID
}

// IssueNoteRequest es el payload común de CreditNote y DebitNote — ambas referencian un
// documento anterior (BillingReference) y comparten todo lo demás con Invoice.
type IssueNoteRequest struct {
	IssuerID            uuid.UUID
	NumberingRangeID    uuid.UUID
	Customer            domain.Party
	Lines               []domain.Line
	PaymentMeans        []domain.PaymentMean
	Note                string
	CurrencyCode        string
	BillingReference    BillingReferenceInput
	DiscrepancyResponse *DiscrepancyResponseInput
	CustomerID          *uuid.UUID // opcional, ver IssueInvoiceRequest.CustomerID
}

// IssueCreditNoteRequest extiende IssueNoteRequest con lo único que CreditNote tiene y
// DebitNote no: el código de concepto de la nota (catálogo DIAN, ver sección 13.2.7.4 del
// Anexo Técnico — no cargado en el seed de catálogos, ver docs/api-dian-architecture.md
// sección 9.6, así que el llamador lo provee directamente).
type IssueCreditNoteRequest struct {
	IssueNoteRequest
	CreditNoteTypeCode string
}

// IssueInvoice construye, firma y — si el ambiente lo permite — envía una Factura
// Electrónica de Venta, persistiendo el resultado en cualquier caso.
//
// El envío real (SendBillSync/SendBillAsync de producción) todavía no existe en cofacture
// (solo SendTestSetAsync, de habilitación) — si el emisor está en producción o el rango no
// tiene un TestSetID, el documento queda construido y firmado (StatusBuilt) sin enviarse; eso
// no es un error de esta llamada, es una limitación conocida (ver
// docs/api-dian-architecture.md sección 9.10).
func (s *Service) IssueInvoice(ctx context.Context, req IssueInvoiceRequest) (*Document, error) {
	if err := validateBase(req.IssuerID, req.NumberingRangeID, req.Lines, req.Customer); err != nil {
		return nil, err
	}

	p, err := s.prepare(ctx, req.IssuerID, req.NumberingRangeID, invoiceDianDocumentType, req.CustomerID)
	if err != nil {
		return nil, err
	}

	currency := defaultCurrency(req.CurrencyCode)
	applyCustomerDefaults(&req.Customer)
	totals := computeTotals(req.Lines)
	headerTaxes := aggregateTaxes(req.Lines)

	inv := domain.Invoice{
		ProfileID:         invoiceProfileID,
		EnvironmentCode:   string(p.iss.Environment),
		OperationTypeCode: invoiceOperationTypeCode,
		DocumentTypeCode:  invoiceDianDocumentType,
		HashType:          invoiceHashType,

		Prefix: p.nr.Prefix,
		Number: strconv.FormatInt(p.number, 10),

		IssueDate: p.now.Format("2006-01-02"),
		IssueTime: p.now.Format("15:04:05-07:00"),

		CurrencyCode: currency,
		Note:         req.Note,

		Supplier: partyFromIssuer(p.iss),
		Customer: req.Customer,

		PaymentMeans: req.PaymentMeans,
		HeaderTaxes:  headerTaxes,
		Totals:       totals,
		Lines:        req.Lines,

		NumberingRange:   numberingRangeFromRange(p.nr),
		SoftwareProvider: softwareProviderFromIssuer(p.iss),
	}

	inv.CUFE = cufe.Compute(inv, p.nr.TechnicalKey)
	inv.SoftwareSecurityCode = securitycode.Compute(p.iss.SoftwareID, p.iss.SoftwarePIN, inv.Prefix+inv.Number)
	inv.QRURL = qr.URL(inv.EnvironmentCode, inv.CUFE)

	doc, err := builder.BuildInvoice(inv)
	if err != nil {
		return nil, fmt.Errorf("construir XML: %w", err)
	}

	partial := Document{
		DianDocumentTypeCode: invoiceDianDocumentType,
		Prefix:               inv.Prefix,
		Number:               p.number,
		DocumentKey:          inv.CUFE,
		IssueDate:            p.now,
		IssueTime:            inv.IssueTime,
		CurrencyCode:         currency,
		Customer:             req.Customer,
		CustomerID:           req.CustomerID,
		Lines:                req.Lines,
		PaymentMeans:         req.PaymentMeans,
		Totals:               totals,
		QRURL:                inv.QRURL,
	}

	return s.finalizeAndSend(ctx, doc, p, partial, zip.KindInvoice)
}

// IssueCreditNote construye, firma y — si el ambiente lo permite — envía una Nota Crédito.
// Reutiliza el mismo pipeline que IssueInvoice, salvo CUDE en vez de CUFE y BuildCreditNote
// en vez de BuildInvoice.
func (s *Service) IssueCreditNote(ctx context.Context, req IssueCreditNoteRequest) (*Document, error) {
	if err := validateNoteRequest(req.IssueNoteRequest); err != nil {
		return nil, err
	}

	p, err := s.prepare(ctx, req.IssuerID, req.NumberingRangeID, creditNoteDianDocumentType, req.CustomerID)
	if err != nil {
		return nil, err
	}

	base := s.buildNoteBase(req.IssueNoteRequest, p, creditNoteProfileID, creditNoteOperationTypeCode, creditNoteHashType, creditNoteDianDocumentType)

	cn := domain.CreditNote{
		Invoice:             base,
		CreditNoteTypeCode:  req.CreditNoteTypeCode,
		BillingReference:    billingReferenceFromInput(req.BillingReference),
		DiscrepancyResponse: discrepancyResponseFromInput(req.DiscrepancyResponse),
	}

	cn.CUFE = cude.Compute(cn.Invoice, p.iss.SoftwarePIN)
	cn.SoftwareSecurityCode = securitycode.Compute(p.iss.SoftwareID, p.iss.SoftwarePIN, cn.Prefix+cn.Number)
	cn.QRURL = qr.URL(cn.EnvironmentCode, cn.CUFE)

	doc, err := builder.BuildCreditNote(cn)
	if err != nil {
		return nil, fmt.Errorf("construir XML: %w", err)
	}

	partial := documentFromNoteBase(cn.Invoice, creditNoteDianDocumentType, p.number, cn.CUFE, req.IssueNoteRequest)
	partial.BillingReference = &req.BillingReference
	partial.DiscrepancyResponse = req.DiscrepancyResponse
	partial.NoteTypeCode = req.CreditNoteTypeCode

	return s.finalizeAndSend(ctx, doc, p, partial, zip.KindCreditNote)
}

// IssueDebitNote construye, firma y — si el ambiente lo permite — envía una Nota Débito.
// A diferencia de CreditNote, DebitNote no tiene un campo de tipo propio en cofacture.
func (s *Service) IssueDebitNote(ctx context.Context, req IssueNoteRequest) (*Document, error) {
	if err := validateNoteRequest(req); err != nil {
		return nil, err
	}

	p, err := s.prepare(ctx, req.IssuerID, req.NumberingRangeID, debitNoteDianDocumentType, req.CustomerID)
	if err != nil {
		return nil, err
	}

	base := s.buildNoteBase(req, p, debitNoteProfileID, debitNoteOperationTypeCode, debitNoteHashType, debitNoteDianDocumentType)

	dn := domain.DebitNote{
		Invoice:             base,
		BillingReference:    billingReferenceFromInput(req.BillingReference),
		DiscrepancyResponse: discrepancyResponseFromInput(req.DiscrepancyResponse),
	}

	dn.CUFE = cude.Compute(dn.Invoice, p.iss.SoftwarePIN)
	dn.SoftwareSecurityCode = securitycode.Compute(p.iss.SoftwareID, p.iss.SoftwarePIN, dn.Prefix+dn.Number)
	dn.QRURL = qr.URL(dn.EnvironmentCode, dn.CUFE)

	doc, err := builder.BuildDebitNote(dn)
	if err != nil {
		return nil, fmt.Errorf("construir XML: %w", err)
	}

	partial := documentFromNoteBase(dn.Invoice, debitNoteDianDocumentType, p.number, dn.CUFE, req)
	partial.BillingReference = &req.BillingReference
	partial.DiscrepancyResponse = req.DiscrepancyResponse

	return s.finalizeAndSend(ctx, doc, p, partial, zip.KindDebitNote)
}

// GetDocument devuelve un documento por ID.
func (s *Service) GetDocument(ctx context.Context, id uuid.UUID) (*Document, error) {
	return s.repo.GetByID(ctx, id)
}

// DefaultListLimit/MaxListLimit acotan ListDocuments — ningún llamador (HTTP o futuro) puede
// pedir una página sin límite.
const (
	DefaultListLimit = 50
	MaxListLimit     = 200
)

// ListDocuments devuelve los documentos de un emisor, opcionalmente filtrados. Limit/Offset
// se normalizan aquí, no en el repositorio: nunca cero/negativos, nunca por encima de
// MaxListLimit.
func (s *Service) ListDocuments(ctx context.Context, issuerID uuid.UUID, filter ListFilter) ([]*Document, error) {
	switch {
	case filter.Limit <= 0:
		filter.Limit = DefaultListLimit
	case filter.Limit > MaxListLimit:
		filter.Limit = MaxListLimit
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.ListByIssuer(ctx, issuerID, filter)
}

// ── Preparación común (emisor, rango, consecutivo, certificado) ───────────────────────────

// preparedIssuance agrupa lo que los tres IssueXxx necesitan antes de construir su propio
// modelo de dominio — carga el emisor y el rango UNA sola vez, valida el tipo de documento,
// reclama el consecutivo, y carga el certificado.
type preparedIssuance struct {
	iss    *issuers.Issuer
	nr     *numbering.NumberingRange
	number int64
	cert   *x509.Certificate
	key    *rsa.PrivateKey
	now    time.Time
}

func (s *Service) prepare(ctx context.Context, issuerID, rangeID uuid.UUID, expectedDianDocType string, customerID *uuid.UUID) (*preparedIssuance, error) {
	iss, err := s.issuers.GetIssuer(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	nr, err := s.numbering.GetRange(ctx, rangeID)
	if err != nil {
		return nil, err
	}
	if nr.IssuerID != issuerID {
		return nil, ErrNumberingRangeIssuerMismatch
	}
	if nr.DianDocumentTypeCode != expectedDianDocType {
		return nil, ErrWrongDocumentType
	}
	if customerID != nil {
		c, err := s.customers.GetCustomer(ctx, *customerID)
		if err != nil {
			return nil, err
		}
		if c.IssuerID != issuerID {
			return nil, ErrCustomerIssuerMismatch
		}
	}

	number, err := s.numbering.ClaimNext(ctx, rangeID)
	if err != nil {
		return nil, err
	}

	cert, key, err := signer.LoadPKCS12(iss.Certificate, iss.CertificatePassword)
	if err != nil {
		return nil, fmt.Errorf("cargar certificado del emisor: %w", err)
	}

	return &preparedIssuance{
		iss: iss, nr: nr, number: number, cert: cert, key: key,
		now: time.Now().In(domain.Bogota),
	}, nil
}

// buildNoteBase construye el domain.Invoice embebido que comparten CreditNote y DebitNote —
// idéntico a IssueInvoice salvo los valores fijos (ProfileID/OperationTypeCode/HashType) y
// que no lleva CUFE/SoftwareSecurityCode/QRURL todavía (eso lo calcula cada llamador con la
// fórmula correcta: cude.Compute en vez de cufe.Compute).
func (s *Service) buildNoteBase(req IssueNoteRequest, p *preparedIssuance, profileID, operationTypeCode, hashType, dianDocType string) domain.Invoice {
	applyCustomerDefaults(&req.Customer)
	currency := defaultCurrency(req.CurrencyCode)
	totals := computeTotals(req.Lines)
	headerTaxes := aggregateTaxes(req.Lines)

	return domain.Invoice{
		ProfileID:         profileID,
		EnvironmentCode:   string(p.iss.Environment),
		OperationTypeCode: operationTypeCode,
		DocumentTypeCode:  dianDocType,
		HashType:          hashType,

		Prefix: p.nr.Prefix,
		Number: strconv.FormatInt(p.number, 10),

		IssueDate: p.now.Format("2006-01-02"),
		IssueTime: p.now.Format("15:04:05-07:00"),

		CurrencyCode: currency,
		Note:         req.Note,

		Supplier: partyFromIssuer(p.iss),
		Customer: req.Customer,

		PaymentMeans: req.PaymentMeans,
		HeaderTaxes:  headerTaxes,
		Totals:       totals,
		Lines:        req.Lines,

		NumberingRange:   numberingRangeFromRange(p.nr),
		SoftwareProvider: softwareProviderFromIssuer(p.iss),
	}
}

// documentFromNoteBase arma el Document parcial (sin SignedXML/Status) a partir del Invoice
// embebido ya construido — comparte exactamente los mismos campos que IssueInvoice persiste.
func documentFromNoteBase(base domain.Invoice, dianDocType string, number int64, documentKey string, req IssueNoteRequest) Document {
	return Document{
		DianDocumentTypeCode: dianDocType,
		Prefix:               base.Prefix,
		Number:               number,
		DocumentKey:          documentKey,
		IssueDate:            time.Time{}, // se completa abajo por el llamador con p.now
		IssueTime:            base.IssueTime,
		CurrencyCode:         base.CurrencyCode,
		Customer:             req.Customer,
		CustomerID:           req.CustomerID,
		Lines:                req.Lines,
		PaymentMeans:         req.PaymentMeans,
		Totals:               base.Totals,
		QRURL:                base.QRURL,
	}
}

// ── Firma, serialización, persistencia y envío (compartido por los tres tipos) ────────────

// finalizeAndSend firma el documento ya construido, lo serializa, lo persiste, y —si el
// ambiente lo permite— lo envía. Common tail de IssueInvoice/IssueCreditNote/IssueDebitNote.
func (s *Service) finalizeAndSend(ctx context.Context, doc *etree.Document, p *preparedIssuance, partial Document, kind zip.DocumentKind) (*Document, error) {
	placeholder, err := builder.SignaturePlaceholder(doc)
	if err != nil {
		return nil, fmt.Errorf("crear placeholder de firma: %w", err)
	}
	if err := signer.New(p.cert, p.key).Sign(doc.Root(), placeholder, "supplier", p.now); err != nil {
		return nil, fmt.Errorf("firmar documento: %w", err)
	}

	// Nunca llamar doc.Indent() (ni nada que reescriba el árbol) después de firmar — ver
	// comentario equivalente en cofacture/soap/realsend_test.go.
	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		return nil, fmt.Errorf("serializar XML firmado: %w", err)
	}

	partial.IssuerID = p.iss.ID
	partial.NumberingRangeID = p.nr.ID
	partial.IssueDate = p.now
	partial.SignedXML = string(xmlBytes)
	partial.Status = StatusBuilt

	created, err := s.repo.Create(ctx, partial)
	if err != nil {
		return nil, err
	}

	// Producción real: pendiente de decisión explícita del usuario (ver sección 9.11 del
	// architecture doc) — se construye y firma, pero no se envía todavía. Se exige que TANTO
	// el emisor COMO el rango estén marcados en habilitación — doble candado a propósito,
	// dado que ahora se envía de verdad incluso sin TestSetID (ver 9.14).
	if p.iss.Environment != issuers.EnvironmentHabilitacion || p.nr.Environment != numbering.EnvironmentHabilitacion {
		return created, nil
	}

	// Con TestSetID: el flujo de certificación (SendTestSetAsync), asíncrono. Sin TestSetID:
	// habilitación sigue aceptando envíos normales aun con el set de pruebas ya cerrado (ver
	// sección 9.14) — SendBillSync, síncrono, un documento a la vez.
	if p.nr.TestSetID != "" {
		return s.sendAndUpdate(ctx, created, p.cert, p.key, p.iss, xmlBytes, p.number, p.now, p.nr.TestSetID, kind)
	}
	return s.sendSyncAndUpdate(ctx, created, p.cert, p.key, p.iss, xmlBytes, p.number, p.now, kind)
}

// sendAndUpdate envía el documento ya firmado a la DIAN (habilitación), sondea el resultado,
// y actualiza el estado persistido. Errores de transporte se reportan como StatusSendError
// (no se pierde el documento ya construido y numerado).
func (s *Service) sendAndUpdate(
	ctx context.Context,
	d *Document,
	cert *x509.Certificate, key *rsa.PrivateKey,
	iss *issuers.Issuer,
	xmlBytes []byte,
	number int64,
	now time.Time,
	testSetID string,
	kind zip.DocumentKind,
) (*Document, error) {
	fileName := zip.DocumentFileName(kind, iss.NIT, zip.SoftwarePropioCode, now.Year(), uint32(number))
	zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
	if err != nil {
		return s.markSendError(ctx, d, fmt.Errorf("comprimir documento: %w", err))
	}
	zipFileName := zip.PackageFileName(iss.NIT, zip.SoftwarePropioCode, now.Year(), uint32(number))

	client := soap.New(soap.HabilitacionURL, cert, key)
	result, err := client.SendTestSetAsync(zipFileName, zipBytes, testSetID)
	if err != nil {
		return s.markSendError(ctx, d, fmt.Errorf("enviar a la DIAN: %w", err))
	}
	if result.ZipKey == "" {
		return s.markSendError(ctx, d, fmt.Errorf("la DIAN rechazó el envío sin ZipKey"))
	}

	d.DianTrackID = result.ZipKey

	// Sondeo acotado — mismo patrón que cofacture/soap/realsend_test.go.
	var last *soap.DianResponse
	for attempt := 0; attempt < 6; attempt++ {
		time.Sleep(5 * time.Second)
		statuses, err := client.GetStatusZip(result.ZipKey)
		if err != nil || len(statuses) == 0 {
			continue
		}
		last = &statuses[len(statuses)-1]
		if last.StatusCode != "" {
			break
		}
	}

	if last == nil {
		return s.finish(ctx, d, StatusSent, result.ZipKey, "", "", "respuesta de la DIAN no disponible todavía (sondeo agotado)")
	}

	interpreted, err := dian.Interpret(*last)
	if err != nil {
		return s.finish(ctx, d, StatusSendError, result.ZipKey, last.StatusCode, "", fmt.Sprintf("interpretar respuesta: %v", err))
	}

	status := StatusAccepted
	if interpreted.HasRejections() || !interpreted.IsValid {
		status = StatusRejected
	}
	return s.finish(ctx, d, status, result.ZipKey, interpreted.StatusCode, interpreted.StatusDescription, interpreted.StatusMessage)
}

// sendSyncAndUpdate envía el documento por SendBillSync — un solo documento, síncrono, sin
// ZipKey ni sondeo posterior: la DIAN responde con el resultado final en la misma llamada.
// Confirmado contra la DIAN real (cofacture/soap/realsend_sync_test.go) que esto sigue
// disponible en habilitación incluso con el set de pruebas oficial ya cerrado — ver
// docs/api-dian-architecture.md sección 9.14.
func (s *Service) sendSyncAndUpdate(
	ctx context.Context,
	d *Document,
	cert *x509.Certificate, key *rsa.PrivateKey,
	iss *issuers.Issuer,
	xmlBytes []byte,
	number int64,
	now time.Time,
	kind zip.DocumentKind,
) (*Document, error) {
	fileName := zip.DocumentFileName(kind, iss.NIT, zip.SoftwarePropioCode, now.Year(), uint32(number))
	zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
	if err != nil {
		return s.markSendError(ctx, d, fmt.Errorf("comprimir documento: %w", err))
	}
	zipFileName := zip.PackageFileName(iss.NIT, zip.SoftwarePropioCode, now.Year(), uint32(number))

	client := soap.New(soap.HabilitacionURL, cert, key)
	resp, err := client.SendBillSync(zipFileName, zipBytes)
	if err != nil {
		return s.markSendError(ctx, d, fmt.Errorf("enviar a la DIAN: %w", err))
	}

	interpreted, err := dian.Interpret(*resp)
	if err != nil {
		return s.finish(ctx, d, StatusSendError, "", resp.StatusCode, "", fmt.Sprintf("interpretar respuesta: %v", err))
	}

	status := StatusAccepted
	if interpreted.HasRejections() || !interpreted.IsValid {
		status = StatusRejected
	}
	return s.finish(ctx, d, status, "", interpreted.StatusCode, interpreted.StatusDescription, interpreted.StatusMessage)
}

func (s *Service) markSendError(ctx context.Context, d *Document, sendErr error) (*Document, error) {
	return s.finish(ctx, d, StatusSendError, "", "", "", sendErr.Error())
}

func (s *Service) finish(ctx context.Context, d *Document, status Status, trackID, statusCode, statusDescription, statusMessage string) (*Document, error) {
	if err := s.repo.UpdateDianStatus(ctx, d.ID, status, trackID, statusCode, statusDescription, statusMessage); err != nil {
		return nil, err
	}
	d.Status = status
	d.DianTrackID = trackID
	d.DianStatusCode = statusCode
	d.DianStatusDescription = statusDescription
	d.DianStatusMessage = statusMessage
	return d, nil
}

// ── Validaciones ───────────────────────────────────────────────────────────────────────────

func validateBase(issuerID, rangeID uuid.UUID, lines []domain.Line, customer domain.Party) error {
	if issuerID == uuid.Nil {
		return ErrMissingIssuer
	}
	if rangeID == uuid.Nil {
		return ErrMissingNumberingRange
	}
	if len(lines) == 0 {
		return ErrEmptyLines
	}
	if customer.Identification.Number == "" {
		return ErrMissingCustomer
	}
	return nil
}

func validateNoteRequest(req IssueNoteRequest) error {
	if err := validateBase(req.IssuerID, req.NumberingRangeID, req.Lines, req.Customer); err != nil {
		return err
	}
	if req.BillingReference.CUFE == "" {
		return ErrMissingBillingReference
	}
	return nil
}

// ── Helpers de construcción de domain.* a partir del emisor/rango ─────────────────────────

// partyFromIssuer construye el cac:AccountingSupplierParty/cac:Party a partir del emisor.
func partyFromIssuer(iss *issuers.Issuer) domain.Party {
	return domain.Party{
		EntityTypeCode: iss.EntityTypeCode,
		Identification: domain.Identification{
			Number:   iss.NIT,
			TypeCode: iss.IdentificationTypeCode,
		},
		Name: iss.BusinessName,
		Address: domain.Address{
			Line:        iss.AddressLine,
			CityCode:    iss.MunicipalityCode,
			CityName:    iss.MunicipalityName,
			StateCode:   iss.DepartmentCode,
			StateName:   iss.DepartmentName,
			CountryCode: "CO",
			CountryName: "Colombia",
		},
		LiabilityCodes:             iss.LiabilityCodes,
		TaxSchemeCode:              iss.TaxSchemeCode,
		TaxSchemeName:              iss.TaxSchemeName,
		Phone:                      iss.Phone,
		Email:                      iss.Email,
		MerchantRegistrationNumber: iss.MerchantRegistrationNumber,
	}
}

func numberingRangeFromRange(nr *numbering.NumberingRange) domain.NumberingRange {
	return domain.NumberingRange{
		AuthorizedCode: nr.ResolutionNumber,
		Prefix:         nr.Prefix,
		StartNumber:    strconv.FormatInt(nr.RangeFrom, 10),
		EndNumber:      rangeToString(nr.RangeTo),
		StartDate:      nr.ValidFrom.Format("2006-01-02"),
		EndDate:        nr.ValidTo.Format("2006-01-02"),
	}
}

func softwareProviderFromIssuer(iss *issuers.Issuer) domain.SoftwareProvider {
	return domain.SoftwareProvider{
		ProviderIdentification: domain.Identification{
			Number: iss.NIT,
			// Siempre "31" (NIT) — la DIAN registra al proveedor de software/facturador
			// electrónico por NIT sin importar el tipo de identificación personal del
			// emisor (ej. un emisor persona natural se identifica como "13" en
			// Supplier.Identification, pero como "31" aquí). Un rechazo real (FAB23 +
			// FAB22b) lo confirmó al reutilizar por error iss.IdentificationTypeCode.
			TypeCode:         "31",
			VerificationCode: iss.CheckDigit,
		},
		SoftwareID: iss.SoftwareID,
	}
}

func billingReferenceFromInput(b BillingReferenceInput) domain.BillingReference {
	return domain.BillingReference{
		Prefix:    b.Prefix,
		Number:    b.Number,
		CUFE:      b.CUFE,
		IssueDate: b.IssueDate,
	}
}

func discrepancyResponseFromInput(d *DiscrepancyResponseInput) *domain.DiscrepancyResponse {
	if d == nil {
		return nil
	}
	return &domain.DiscrepancyResponse{
		ReferenceID:  d.ReferenceID,
		ResponseCode: d.ResponseCode,
		Description:  d.Description,
	}
}

// applyCustomerDefaults completa lo que la mayoría de adquirientes no necesita personalizar
// — mismo criterio que issuers.applyDefaults, valores confirmados contra la DIAN real.
func applyCustomerDefaults(p *domain.Party) {
	if p.EntityTypeCode == "" {
		p.EntityTypeCode = "2"
	}
	if p.TaxSchemeCode == "" {
		p.TaxSchemeCode = "ZZ"
	}
	if p.TaxSchemeName == "" {
		p.TaxSchemeName = "No aplica"
	}
	if len(p.LiabilityCodes) == 0 {
		p.LiabilityCodes = []string{"R-99-PN"}
	}
}

func defaultCurrency(c string) string {
	if c == "" {
		return "COP"
	}
	return c
}

// computeTotals suma las líneas — no se le pide al llamador que lo calcule y posiblemente
// lo haga mal o inconsistente con las líneas reales.
func computeTotals(lines []domain.Line) domain.Totals {
	var lineExt, taxAmount int64
	for _, l := range lines {
		lineExt += l.LineExtensionCents
		for _, t := range l.Taxes {
			taxAmount += t.TaxAmountCents
		}
	}
	taxInclusive := lineExt + taxAmount
	return domain.Totals{
		LineExtensionCents: lineExt,
		TaxExclusiveCents:  lineExt,
		TaxInclusiveCents:  taxInclusive,
		PayableCents:       taxInclusive,
	}
}

// aggregateTaxes agrupa los impuestos de todas las líneas por TypeCode para construir
// cac:TaxTotal de cabecera — la DIAN exige que la base imponible total coincida con la suma
// de las bases de las líneas (regla FAU04, ver comentario en realsend_test.go).
func aggregateTaxes(lines []domain.Line) []domain.Tax {
	type agg struct {
		typeName           string
		taxableAmountCents int64
		taxAmountCents     int64
		percent            float64
	}
	byCode := make(map[string]*agg)
	var order []string

	for _, l := range lines {
		for _, t := range l.Taxes {
			a, ok := byCode[t.TypeCode]
			if !ok {
				a = &agg{typeName: t.TypeName, percent: t.Percent}
				byCode[t.TypeCode] = a
				order = append(order, t.TypeCode)
			}
			a.taxableAmountCents += t.TaxableAmountCents
			a.taxAmountCents += t.TaxAmountCents
		}
	}

	taxes := make([]domain.Tax, 0, len(order))
	for _, code := range order {
		a := byCode[code]
		taxes = append(taxes, domain.Tax{
			TaxableAmountCents: a.taxableAmountCents,
			TaxAmountCents:     a.taxAmountCents,
			Percent:            a.percent,
			TypeCode:           code,
			TypeName:           a.typeName,
		})
	}
	return taxes
}

func rangeToString(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}
