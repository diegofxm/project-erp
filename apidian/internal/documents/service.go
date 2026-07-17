package documents

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"strconv"
	"time"

	"github.com/beevik/etree"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/cude"
	"github.com/diegofxm/cofacture/cuds"
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

	supportDocumentProfileID    = "DIAN 2.1: documento soporte en adquisiciones efectuadas a no obligados a facturar."
	supportDocumentHashType     = "CUDS-SHA384"
	supportDocumentDianDocType  = "05"
)

// Service orquesta el ciclo de vida de documentos DIAN: crear borradores, editarlos,
// confirmarlos (reclamar numeración, construir y firmar el XML con cofacture, enviar por
// SOAP, interpretar la respuesta), y persistir el resultado.
//
// Es el ÚNICO paquete de apidian que importa cofacture directamente — ver
// docs/apidian-architecture.md sección 4.1.
type Service struct {
	repo      Repository
	issuers   IssuerPort
	numbering NumberingPort
	customers CustomerPort
	vendors   VendorPort // opcional; nil si no se configuró el catálogo de proveedores
	catalogs  CatalogPort
	email     EmailPort
}

// New crea el servicio de documentos.
func New(repo Repository, issuerPort IssuerPort, numberingPort NumberingPort, customerPort CustomerPort, catalogPort CatalogPort, emailPort EmailPort) *Service {
	return &Service{repo: repo, issuers: issuerPort, numbering: numberingPort, customers: customerPort, catalogs: catalogPort, email: emailPort}
}

// WithVendors establece el puerto de proveedores para validar VendorID en Documentos Soporte.
// Se llama en cadena sobre New() desde api.go; los tests que no necesitan vendors lo omiten.
func (s *Service) WithVendors(p VendorPort) *Service {
	s.vendors = p
	return s
}

// IssueInvoiceRequest es el payload de una Factura Electrónica de Venta — sirve tanto para
// crear el borrador como para reemplazarlo por completo en una actualización. Customer/Lines/
// PaymentMeans son pass-through: llegan tal cual y se persisten como snapshot (ver model.go).
type IssueInvoiceRequest struct {
	IssuerID         uuid.UUID
	NumberingRangeID uuid.UUID
	Customer         domain.Party
	Lines            []LineInput
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
	Lines               []LineInput
	PaymentMeans        []domain.PaymentMean
	Note                string
	CurrencyCode        string
	BillingReference    BillingReferenceInput
	DiscrepancyResponse *DiscrepancyResponseInput
	CustomerID          *uuid.UUID // opcional, ver IssueInvoiceRequest.CustomerID
}

// IssueCreditNoteRequest extiende IssueNoteRequest con lo único que CreditNote tiene y
// DebitNote no: el código de concepto de la nota (catálogo DIAN, ver sección 13.2.7.4 del
// Anexo Técnico — no cargado en el seed de catálogos, ver docs/apidian-architecture.md
// sección 9.6, así que el llamador lo provee directamente).
type IssueCreditNoteRequest struct {
	IssueNoteRequest
	CreditNoteTypeCode string
}

// IssueSupportDocumentRequest es el payload de un Documento Soporte (InvoiceTypeCode "05").
//
// Los roles están invertidos respecto a la Factura: Vendor es el tercero no obligado
// (AccountingSupplierParty), y la empresa emisora actúa como compradora
// (AccountingCustomerParty — no se recibe del cliente, se deriva del emisor al confirmar).
// OperationTypeCode distingue si el vendedor es Residente ("10") o No Residente ("11").
// WithholdingTaxes son las retenciones calculadas (ReteIVA código "05", ReteRenta código "06").
type IssueSupportDocumentRequest struct {
	IssuerID          uuid.UUID
	NumberingRangeID  uuid.UUID
	Vendor            domain.Party
	Lines             []LineInput
	PaymentMeans      []domain.PaymentMean
	Note              string
	CurrencyCode      string
	OperationTypeCode string    // "10" Residente, "11" No Residente
	WithholdingTaxes  []domain.Tax
	VendorID          *uuid.UUID // opcional, trazabilidad al catálogo de proveedores
}

// ── Crear / editar / eliminar borradores ────────────────────────────────────────────────────
//
// Ningún CreateXDraft/UpdateXDraft reclama número, firma ni envía — eso solo pasa en
// ConfirmDocument. Así un error de captura no quema un consecutivo real de la DIAN (ver
// docs/apidian-architecture.md sección 9.25).

// CreateInvoiceDraft valida y persiste un borrador de Factura Electrónica de Venta.
func (s *Service) CreateInvoiceDraft(ctx context.Context, req IssueInvoiceRequest) (*Document, error) {
	if err := s.validateBase(ctx, req.IssuerID, req.NumberingRangeID, req.Lines, req.Customer, req.PaymentMeans); err != nil {
		return nil, err
	}
	if _, err := s.validateForIssuance(ctx, req.IssuerID, req.NumberingRangeID, invoiceDianDocumentType, req.CustomerID); err != nil {
		return nil, err
	}
	applyCustomerDefaults(&req.Customer)
	if err := s.resolveCustomerTaxSchemeName(ctx, &req.Customer); err != nil {
		return nil, err
	}
	lines, err := s.linesFromInput(ctx, req.Lines)
	if err != nil {
		return nil, err
	}

	draft := Document{
		IssuerID:             req.IssuerID,
		NumberingRangeID:     req.NumberingRangeID,
		DianDocumentTypeCode: invoiceDianDocumentType,
		CurrencyCode:         defaultCurrency(req.CurrencyCode),
		Note:                 req.Note,
		Customer:             req.Customer,
		CustomerID:           req.CustomerID,
		Lines:                lines,
		PaymentMeans:         req.PaymentMeans,
		Totals:               computeTotals(lines),
		Status:               StatusDraft,
	}
	return s.repo.Create(ctx, draft)
}

// UpdateInvoiceDraft reemplaza por completo los datos de un borrador de Factura existente —
// solo mientras siga en borrador (ErrDocumentNotDraft si no) y solo si pertenece al emisor
// dado (ErrDocumentNotFound si no, mismo criterio "indistinguible de no-existe" que el resto
// de la API).
func (s *Service) UpdateInvoiceDraft(ctx context.Context, id uuid.UUID, req IssueInvoiceRequest) (*Document, error) {
	if err := s.validateBase(ctx, req.IssuerID, req.NumberingRangeID, req.Lines, req.Customer, req.PaymentMeans); err != nil {
		return nil, err
	}
	if err := s.requireOwnDraft(ctx, req.IssuerID, id); err != nil {
		return nil, err
	}
	if _, err := s.validateForIssuance(ctx, req.IssuerID, req.NumberingRangeID, invoiceDianDocumentType, req.CustomerID); err != nil {
		return nil, err
	}
	applyCustomerDefaults(&req.Customer)
	if err := s.resolveCustomerTaxSchemeName(ctx, &req.Customer); err != nil {
		return nil, err
	}
	lines, err := s.linesFromInput(ctx, req.Lines)
	if err != nil {
		return nil, err
	}

	draft := Document{
		ID:                   id,
		IssuerID:             req.IssuerID,
		NumberingRangeID:     req.NumberingRangeID,
		DianDocumentTypeCode: invoiceDianDocumentType,
		CurrencyCode:         defaultCurrency(req.CurrencyCode),
		Note:                 req.Note,
		Customer:             req.Customer,
		CustomerID:           req.CustomerID,
		Lines:                lines,
		PaymentMeans:         req.PaymentMeans,
		Totals:               computeTotals(lines),
		Status:               StatusDraft,
	}
	return s.repo.UpdateDraft(ctx, draft)
}

// CreateCreditNoteDraft valida y persiste un borrador de Nota Crédito.
func (s *Service) CreateCreditNoteDraft(ctx context.Context, req IssueCreditNoteRequest) (*Document, error) {
	if err := s.validateNoteRequest(ctx, req.IssueNoteRequest); err != nil {
		return nil, err
	}
	if _, err := s.validateForIssuance(ctx, req.IssuerID, req.NumberingRangeID, creditNoteDianDocumentType, req.CustomerID); err != nil {
		return nil, err
	}
	draft, err := s.noteDraftFromRequest(ctx, req.IssueNoteRequest, creditNoteDianDocumentType)
	if err != nil {
		return nil, err
	}
	draft.NoteTypeCode = req.CreditNoteTypeCode
	return s.repo.Create(ctx, draft)
}

// UpdateCreditNoteDraft reemplaza por completo los datos de un borrador de Nota Crédito
// existente — mismas reglas que UpdateInvoiceDraft.
func (s *Service) UpdateCreditNoteDraft(ctx context.Context, id uuid.UUID, req IssueCreditNoteRequest) (*Document, error) {
	if err := s.validateNoteRequest(ctx, req.IssueNoteRequest); err != nil {
		return nil, err
	}
	if err := s.requireOwnDraft(ctx, req.IssuerID, id); err != nil {
		return nil, err
	}
	if _, err := s.validateForIssuance(ctx, req.IssuerID, req.NumberingRangeID, creditNoteDianDocumentType, req.CustomerID); err != nil {
		return nil, err
	}
	draft, err := s.noteDraftFromRequest(ctx, req.IssueNoteRequest, creditNoteDianDocumentType)
	if err != nil {
		return nil, err
	}
	draft.ID = id
	draft.NoteTypeCode = req.CreditNoteTypeCode
	return s.repo.UpdateDraft(ctx, draft)
}

// CreateDebitNoteDraft valida y persiste un borrador de Nota Débito.
func (s *Service) CreateDebitNoteDraft(ctx context.Context, req IssueNoteRequest) (*Document, error) {
	if err := s.validateNoteRequest(ctx, req); err != nil {
		return nil, err
	}
	if _, err := s.validateForIssuance(ctx, req.IssuerID, req.NumberingRangeID, debitNoteDianDocumentType, req.CustomerID); err != nil {
		return nil, err
	}
	draft, err := s.noteDraftFromRequest(ctx, req, debitNoteDianDocumentType)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, draft)
}

// UpdateDebitNoteDraft reemplaza por completo los datos de un borrador de Nota Débito
// existente — mismas reglas que UpdateInvoiceDraft.
func (s *Service) UpdateDebitNoteDraft(ctx context.Context, id uuid.UUID, req IssueNoteRequest) (*Document, error) {
	if err := s.validateNoteRequest(ctx, req); err != nil {
		return nil, err
	}
	if err := s.requireOwnDraft(ctx, req.IssuerID, id); err != nil {
		return nil, err
	}
	if _, err := s.validateForIssuance(ctx, req.IssuerID, req.NumberingRangeID, debitNoteDianDocumentType, req.CustomerID); err != nil {
		return nil, err
	}
	draft, err := s.noteDraftFromRequest(ctx, req, debitNoteDianDocumentType)
	if err != nil {
		return nil, err
	}
	draft.ID = id
	return s.repo.UpdateDraft(ctx, draft)
}

// CreateSupportDocumentDraft valida y persiste un borrador de Documento Soporte.
func (s *Service) CreateSupportDocumentDraft(ctx context.Context, req IssueSupportDocumentRequest) (*Document, error) {
	if err := s.validateSupportDocumentRequest(ctx, req); err != nil {
		return nil, err
	}
	if _, err := s.validateForIssuance(ctx, req.IssuerID, req.NumberingRangeID, supportDocumentDianDocType, nil); err != nil {
		return nil, err
	}
	if err := s.validateVendorID(ctx, req.IssuerID, req.VendorID); err != nil {
		return nil, err
	}
	applyVendorDefaults(&req.Vendor)
	lines, err := s.linesFromInput(ctx, req.Lines)
	if err != nil {
		return nil, err
	}
	draft := s.supportDocumentDraftFromRequest(req, lines)
	return s.repo.Create(ctx, draft)
}

// UpdateSupportDocumentDraft reemplaza por completo los datos de un borrador de Documento
// Soporte existente — mismas reglas que UpdateInvoiceDraft.
func (s *Service) UpdateSupportDocumentDraft(ctx context.Context, id uuid.UUID, req IssueSupportDocumentRequest) (*Document, error) {
	if err := s.validateSupportDocumentRequest(ctx, req); err != nil {
		return nil, err
	}
	if err := s.requireOwnDraft(ctx, req.IssuerID, id); err != nil {
		return nil, err
	}
	if _, err := s.validateForIssuance(ctx, req.IssuerID, req.NumberingRangeID, supportDocumentDianDocType, nil); err != nil {
		return nil, err
	}
	if err := s.validateVendorID(ctx, req.IssuerID, req.VendorID); err != nil {
		return nil, err
	}
	applyVendorDefaults(&req.Vendor)
	lines, err := s.linesFromInput(ctx, req.Lines)
	if err != nil {
		return nil, err
	}
	draft := s.supportDocumentDraftFromRequest(req, lines)
	draft.ID = id
	return s.repo.UpdateDraft(ctx, draft)
}

// validateVendorID verifica que el VendorID opcional pertenezca al mismo emisor — solo si el
// VendorPort está configurado y el caller mandó un VendorID.
func (s *Service) validateVendorID(ctx context.Context, issuerID uuid.UUID, vendorID *uuid.UUID) error {
	if vendorID == nil || s.vendors == nil {
		return nil
	}
	v, err := s.vendors.GetVendor(ctx, *vendorID)
	if err != nil {
		return err
	}
	if v.IssuerID != issuerID {
		return ErrVendorIssuerMismatch
	}
	return nil
}

func (s *Service) supportDocumentDraftFromRequest(req IssueSupportDocumentRequest, lines []domain.Line) Document {
	vendor := req.Vendor
	return Document{
		IssuerID:             req.IssuerID,
		NumberingRangeID:     req.NumberingRangeID,
		DianDocumentTypeCode: supportDocumentDianDocType,
		CurrencyCode:         defaultCurrency(req.CurrencyCode),
		Note:                 req.Note,
		Lines:                lines,
		PaymentMeans:         req.PaymentMeans,
		Totals:               computeTotalsForDS(lines, req.WithholdingTaxes),
		Vendor:               &vendor,
		VendorID:             req.VendorID,
		OperationTypeCode:    req.OperationTypeCode,
		WithholdingTaxes:     req.WithholdingTaxes,
		Status:               StatusDraft,
	}
}

// validateSupportDocumentRequest valida los campos obligatorios del Documento Soporte.
// No reutiliza validateBase porque DS no tiene Customer (la empresa es el comprador) y las
// retenciones son opcionales (no todos los DS las llevan).
func (s *Service) validateSupportDocumentRequest(ctx context.Context, req IssueSupportDocumentRequest) error {
	if req.IssuerID == uuid.Nil {
		return ErrMissingIssuer
	}
	if req.NumberingRangeID == uuid.Nil {
		return ErrMissingNumberingRange
	}
	if len(req.Lines) == 0 {
		return ErrEmptyLines
	}
	if req.Vendor.Identification.Number == "" {
		return ErrMissingCustomer // reuse: "el tercero" es obligatorio igual que el cliente en Invoice
	}
	if req.OperationTypeCode != "10" && req.OperationTypeCode != "11" {
		return fmt.Errorf("%w: operation_type_code debe ser '10' (Residente) o '11' (No Residente)", ErrInvalidPaymentTerm)
	}
	if len(req.PaymentMeans) == 0 {
		return ErrMissingPaymentMeans
	}
	for _, pm := range req.PaymentMeans {
		ok, err := s.catalogs.IsValidPaymentTerm(ctx, pm.Code)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %q", ErrInvalidPaymentTerm, pm.Code)
		}
		ok, err = s.catalogs.IsValidPaymentMethod(ctx, pm.PaymentMethodCode)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %q", ErrInvalidPaymentMethod, pm.PaymentMethodCode)
		}
	}
	return nil
}

// applyVendorDefaults completa los campos que la mayoría de terceros no obligados no
// personalizan — persona natural sin obligación de facturar (código O-49).
func applyVendorDefaults(p *domain.Party) {
	if p.EntityTypeCode == "" {
		p.EntityTypeCode = "1" // persona jurídica (cbc:AdditionalAccountID — confirmado aceptado por DIAN para terceros no obligados)
	}
	if p.TaxSchemeCode == "" {
		p.TaxSchemeCode = "ZZ"
	}
	if p.TaxSchemeName == "" {
		p.TaxSchemeName = "No aplica"
	}
	if len(p.LiabilityCodes) == 0 {
		p.LiabilityCodes = []string{"O-49"}
	}
	if p.TaxRegimeCode == "" {
		p.TaxRegimeCode = "49"
	}
	if p.Address.CountryCode == "" {
		p.Address.CountryCode = "CO"
		p.Address.CountryName = "Colombia"
	}
}

// computeTotalsForDS calcula los totales de un Documento Soporte, descontando las
// retenciones del monto pagable (el PayableCents de DS es neto de retenciones).
func computeTotalsForDS(lines []domain.Line, withholdingTaxes []domain.Tax) domain.Totals {
	var lineExt, lineTaxAmount, withholdingAmount int64
	for _, l := range lines {
		lineExt += l.LineExtensionCents
		for _, t := range l.Taxes {
			lineTaxAmount += t.TaxAmountCents
		}
	}
	for _, t := range withholdingTaxes {
		withholdingAmount += t.TaxAmountCents
	}
	taxInclusive := lineExt + lineTaxAmount
	return domain.Totals{
		LineExtensionCents: lineExt,
		TaxExclusiveCents:  lineExt,
		TaxInclusiveCents:  taxInclusive,
		PayableCents:       taxInclusive - withholdingAmount,
	}
}

// DeleteDraft elimina un borrador — solo mientras siga en borrador y pertenezca al emisor dado.
func (s *Service) DeleteDraft(ctx context.Context, issuerID, id uuid.UUID) error {
	if err := s.requireOwnDraft(ctx, issuerID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// requireOwnDraft centraliza el chequeo que Update*Draft/DeleteDraft repiten: el documento
// existe, pertenece a issuerID (si no, ErrDocumentNotFound — mismo 404 indistinguible que el
// resto de la API), y sigue en borrador (si no, ErrDocumentNotDraft).
func (s *Service) requireOwnDraft(ctx context.Context, issuerID, id uuid.UUID) error {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d.IssuerID != issuerID {
		return ErrDocumentNotFound
	}
	if d.Status != StatusDraft {
		return ErrDocumentNotDraft
	}
	return nil
}

func (s *Service) noteDraftFromRequest(ctx context.Context, req IssueNoteRequest, dianDocType string) (Document, error) {
	applyCustomerDefaults(&req.Customer)
	if err := s.resolveCustomerTaxSchemeName(ctx, &req.Customer); err != nil {
		return Document{}, err
	}
	lines, err := s.linesFromInput(ctx, req.Lines)
	if err != nil {
		return Document{}, err
	}

	d := Document{
		IssuerID:             req.IssuerID,
		NumberingRangeID:     req.NumberingRangeID,
		DianDocumentTypeCode: dianDocType,
		CurrencyCode:         defaultCurrency(req.CurrencyCode),
		Note:                 req.Note,
		Customer:             req.Customer,
		CustomerID:           req.CustomerID,
		Lines:                lines,
		PaymentMeans:         req.PaymentMeans,
		Totals:               computeTotals(lines),
		Status:               StatusDraft,
	}
	billingRef := billingReferenceInputCopy(req.BillingReference)
	d.BillingReference = &billingRef
	d.DiscrepancyResponse = req.DiscrepancyResponse
	return d, nil
}

func billingReferenceInputCopy(b BillingReferenceInput) BillingReferenceInput {
	return b
}

// ── Confirmar (reclamar número, construir, firmar, enviar) ─────────────────────────────────

// ConfirmDocument reclama el consecutivo real, construye y firma el XML, y —si el ambiente lo
// permite— lo envía a la DIAN, para un borrador ya creado. Es el ÚNICO punto de todo el
// servicio donde se "gasta" un número real — antes de esto, el documento se puede editar o
// eliminar libremente.
func (s *Service) ConfirmDocument(ctx context.Context, issuerID, id uuid.UUID) (*Document, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.IssuerID != issuerID {
		return nil, ErrDocumentNotFound
	}
	if d.Status != StatusDraft {
		return nil, ErrDocumentNotDraft
	}

	switch d.DianDocumentTypeCode {
	case invoiceDianDocumentType:
		return s.confirmInvoice(ctx, d)
	case creditNoteDianDocumentType:
		return s.confirmCreditNote(ctx, d)
	case debitNoteDianDocumentType:
		return s.confirmDebitNote(ctx, d)
	case supportDocumentDianDocType:
		return s.confirmSupportDocument(ctx, d)
	default:
		return nil, fmt.Errorf("documents: tipo de documento desconocido %q", d.DianDocumentTypeCode)
	}
}

func (s *Service) confirmInvoice(ctx context.Context, d *Document) (*Document, error) {
	p, err := s.claimAndLoadCert(ctx, d)
	if err != nil {
		return nil, err
	}

	headerTaxes := aggregateTaxes(d.Lines)
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

		CurrencyCode: d.CurrencyCode,
		Note:         d.Note,

		Supplier: partyFromIssuer(p.iss),
		Customer: d.Customer,

		PaymentMeans: d.PaymentMeans,
		HeaderTaxes:  headerTaxes,
		Totals:       d.Totals,
		Lines:        d.Lines,

		NumberingRange:   numberingRangeFromRange(p.nr),
		SoftwareProvider: softwareProviderFromIssuer(p.iss),
	}

	inv.CUFE = cufe.Compute(inv, p.nr.TechnicalKey)
	inv.SoftwareSecurityCode = securitycode.Compute(p.iss.SoftwareID, p.iss.SoftwarePIN, inv.Prefix+inv.Number)
	inv.QRURL = qr.URL(inv.EnvironmentCode, inv.CUFE)

	xmlDoc, err := builder.BuildInvoice(inv)
	if err != nil {
		_ = s.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("construir XML: %w", err)
	}

	d.Prefix, d.Number, d.DocumentKey = inv.Prefix, p.number, inv.CUFE
	d.IssueDate, d.IssueTime, d.QRURL = p.now, inv.IssueTime, inv.QRURL

	return s.finalizeAndSend(ctx, xmlDoc, p, d, zip.KindInvoice)
}

func (s *Service) confirmCreditNote(ctx context.Context, d *Document) (*Document, error) {
	p, err := s.claimAndLoadCert(ctx, d)
	if err != nil {
		return nil, err
	}

	base := s.noteBaseFromDraft(d, p, creditNoteProfileID, creditNoteOperationTypeCode, creditNoteHashType, creditNoteDianDocumentType)
	cn := domain.CreditNote{
		Invoice:             base,
		CreditNoteTypeCode:  creditNoteDianDocumentType, // "91" — tipo DIAN de NC (fijo); el concepto de List 22 va en DiscrepancyResponse.ResponseCode
		BillingReference:    billingReferenceFromInput(*d.BillingReference),
		DiscrepancyResponse: discrepancyResponseFromInput(d.DiscrepancyResponse),
	}

	cn.CUFE = cude.Compute(cn.Invoice, p.iss.SoftwarePIN)
	cn.SoftwareSecurityCode = securitycode.Compute(p.iss.SoftwareID, p.iss.SoftwarePIN, cn.Prefix+cn.Number)
	cn.QRURL = qr.URL(cn.EnvironmentCode, cn.CUFE)

	xmlDoc, err := builder.BuildCreditNote(cn)
	if err != nil {
		_ = s.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("construir XML: %w", err)
	}

	d.Prefix, d.Number, d.DocumentKey = cn.Prefix, p.number, cn.CUFE
	d.IssueDate, d.IssueTime, d.QRURL = p.now, cn.IssueTime, cn.QRURL

	return s.finalizeAndSend(ctx, xmlDoc, p, d, zip.KindCreditNote)
}

func (s *Service) confirmDebitNote(ctx context.Context, d *Document) (*Document, error) {
	p, err := s.claimAndLoadCert(ctx, d)
	if err != nil {
		return nil, err
	}

	base := s.noteBaseFromDraft(d, p, debitNoteProfileID, debitNoteOperationTypeCode, debitNoteHashType, debitNoteDianDocumentType)
	dn := domain.DebitNote{
		Invoice:             base,
		BillingReference:    billingReferenceFromInput(*d.BillingReference),
		DiscrepancyResponse: discrepancyResponseFromInput(d.DiscrepancyResponse),
	}

	dn.CUFE = cude.Compute(dn.Invoice, p.iss.SoftwarePIN)
	dn.SoftwareSecurityCode = securitycode.Compute(p.iss.SoftwareID, p.iss.SoftwarePIN, dn.Prefix+dn.Number)
	dn.QRURL = qr.URL(dn.EnvironmentCode, dn.CUFE)

	xmlDoc, err := builder.BuildDebitNote(dn)
	if err != nil {
		_ = s.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("construir XML: %w", err)
	}

	d.Prefix, d.Number, d.DocumentKey = dn.Prefix, p.number, dn.CUFE
	d.IssueDate, d.IssueTime, d.QRURL = p.now, dn.IssueTime, dn.QRURL

	return s.finalizeAndSend(ctx, xmlDoc, p, d, zip.KindDebitNote)
}

func (s *Service) confirmSupportDocument(ctx context.Context, d *Document) (*Document, error) {
	if d.Vendor == nil {
		return nil, fmt.Errorf("documents: documento soporte sin vendor")
	}
	p, err := s.claimAndLoadCert(ctx, d)
	if err != nil {
		return nil, err
	}

	headerTaxes := aggregateTaxes(d.Lines)
	inv := domain.Invoice{
		ProfileID:         supportDocumentProfileID,
		EnvironmentCode:   string(p.iss.Environment),
		OperationTypeCode: d.OperationTypeCode,
		DocumentTypeCode:  supportDocumentDianDocType,
		HashType:          supportDocumentHashType,

		Prefix: p.nr.Prefix,
		Number: strconv.FormatInt(p.number, 10),

		IssueDate: p.now.Format("2006-01-02"),
		IssueTime: p.now.Format("15:04:05-07:00"),

		CurrencyCode: d.CurrencyCode,
		Note:         d.Note,

		// Roles invertidos: Supplier = tercero no obligado, Customer = empresa compradora.
		Supplier: *d.Vendor,
		Customer: partyFromIssuer(p.iss),

		PaymentMeans:     d.PaymentMeans,
		HeaderTaxes:      headerTaxes,
		WithholdingTaxes: d.WithholdingTaxes,
		Totals:           d.Totals,
		Lines:            d.Lines,

		NumberingRange:   numberingRangeFromRange(p.nr),
		SoftwareProvider: softwareProviderFromIssuer(p.iss),
	}

	inv.CUFE = cuds.Compute(inv, p.iss.SoftwarePIN)
	inv.SoftwareSecurityCode = securitycode.Compute(p.iss.SoftwareID, p.iss.SoftwarePIN, inv.Prefix+inv.Number)
	inv.QRURL = qr.SupportDocumentContent(inv, inv.CUFE, p.iss.SoftwarePIN)

	xmlDoc, err := builder.BuildSupportDocument(inv)
	if err != nil {
		_ = s.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("construir XML: %w", err)
	}

	d.Prefix, d.Number, d.DocumentKey = inv.Prefix, p.number, inv.CUFE
	d.IssueDate, d.IssueTime, d.QRURL = p.now, inv.IssueTime, inv.QRURL

	return s.finalizeAndSend(ctx, xmlDoc, p, d, zip.KindSupportDocument)
}

// noteBaseFromDraft construye el domain.Invoice embebido que comparten CreditNote y
// DebitNote, a partir de un borrador ya persistido — equivalente a lo que antes era
// buildNoteBase a partir de la petición HTTP directamente.
func (s *Service) noteBaseFromDraft(d *Document, p *preparedIssuance, profileID, operationTypeCode, hashType, dianDocType string) domain.Invoice {
	headerTaxes := aggregateTaxes(d.Lines)
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

		CurrencyCode: d.CurrencyCode,
		Note:         d.Note,

		Supplier: partyFromIssuer(p.iss),
		Customer: d.Customer,

		PaymentMeans: d.PaymentMeans,
		HeaderTaxes:  headerTaxes,
		Totals:       d.Totals,
		Lines:        d.Lines,

		NumberingRange:   numberingRangeFromRange(p.nr),
		SoftwareProvider: softwareProviderFromIssuer(p.iss),
	}
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

// preparedIssuance agrupa lo que confirmInvoice/confirmCreditNote/confirmDebitNote necesitan.
// validateForIssuance llena iss/nr (se puede validar en el borrador, sin gastar nada);
// claimAndLoadCert llena number/cert/key/now (solo al confirmar, gasta un número real).
type preparedIssuance struct {
	iss    *issuers.Issuer
	nr     *numbering.NumberingRange
	number int64
	cert   *x509.Certificate
	key    *rsa.PrivateKey
	now    time.Time
}

// validateForIssuance carga emisor + rango y valida que el rango pertenezca a este emisor, le
// corresponda al tipo de documento esperado, y que el CustomerID opcional (si lo hay) también
// pertenezca a este emisor — todo verificable sin reclamar ningún número, por eso se corre
// tanto al crear/editar un borrador como, de nuevo, al confirmarlo.
func (s *Service) validateForIssuance(ctx context.Context, issuerID, rangeID uuid.UUID, expectedDianDocType string, customerID *uuid.UUID) (*preparedIssuance, error) {
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
	return &preparedIssuance{iss: iss, nr: nr}, nil
}

// claimAndLoadCert es el paso que de verdad "gasta" un número: re-valida (defensivo, en caso
// de que algo cambiara entre crear el borrador y confirmarlo), exige que el emisor ya tenga
// software/certificado configurados (ErrIssuerNotReadyToIssue si no — ver
// docs/apidian-architecture.md sección 9.25 sobre configuración gradual del emisor), reclama
// el consecutivo, y carga el certificado para firmar.
func (s *Service) claimAndLoadCert(ctx context.Context, d *Document) (*preparedIssuance, error) {
	p, err := s.validateForIssuance(ctx, d.IssuerID, d.NumberingRangeID, d.DianDocumentTypeCode, d.CustomerID)
	if err != nil {
		return nil, err
	}
	if p.iss.SoftwareID == "" || p.iss.SoftwarePIN == "" || len(p.iss.Certificate) == 0 {
		return nil, ErrIssuerNotReadyToIssue
	}

	number, err := s.numbering.ClaimNext(ctx, d.NumberingRangeID)
	if err != nil {
		return nil, err
	}

	cert, key, err := signer.LoadPKCS12(p.iss.Certificate, p.iss.CertificatePassword)
	if err != nil {
		// El número ya se reclamó pero el documento ni siquiera llegó a construirse — nunca
		// existió ante la DIAN, así que se devuelve (mismo mecanismo y mismo motivo que en
		// finish, ver sección 9.33) en vez de quemarlo por un error de certificado.
		_ = s.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, number)
		return nil, fmt.Errorf("cargar certificado del emisor: %w", err)
	}

	p.number = number
	p.cert = cert
	p.key = key
	p.now = time.Now().In(domain.Bogota)
	return p, nil
}

// ── Firma, serialización, persistencia y envío (compartido por los tres tipos) ────────────

// finalizeAndSend firma el documento ya construido, lo serializa, persiste la confirmación
// del borrador (d.ID ya existía desde la creación), y —si el ambiente lo permite— lo envía.
// Common tail de confirmInvoice/confirmCreditNote/confirmDebitNote.
func (s *Service) finalizeAndSend(ctx context.Context, doc *etree.Document, p *preparedIssuance, d *Document, kind zip.DocumentKind) (*Document, error) {
	placeholder, err := builder.SignaturePlaceholder(doc)
	if err != nil {
		_ = s.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("crear placeholder de firma: %w", err)
	}
	if err := signer.New(p.cert, p.key).Sign(doc.Root(), placeholder, "supplier", p.now); err != nil {
		_ = s.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("firmar documento: %w", err)
	}

	// Nunca llamar doc.Indent() (ni nada que reescriba el árbol) después de firmar — ver
	// comentario equivalente en cofacture/soap/realsend_test.go.
	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		_ = s.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("serializar XML firmado: %w", err)
	}

	d.SignedXML = string(xmlBytes)
	d.Status = StatusBuilt

	created, err := s.repo.Confirm(ctx, *d)
	if err != nil {
		_ = s.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
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
		return s.finish(ctx, d, StatusSent, result.ZipKey, "", "", "respuesta de la DIAN no disponible todavía (sondeo agotado)", "")
	}

	interpreted, err := dian.Interpret(*last)
	if err != nil {
		return s.finish(ctx, d, StatusSendError, result.ZipKey, last.StatusCode, "", fmt.Sprintf("interpretar respuesta: %v", err), "")
	}

	// El test_set_id de este rango ya quedó certificado/cerrado del lado de la DIAN (sección
	// 9.43) — no es un rechazo del contenido del documento, así que no se trata como tal.
	// Se limpia para que ningún confirm futuro vuelva a intentar este camino, y ESTE mismo
	// envío se reintenta de inmediato por SendBillSync (que sigue funcionando con el set
	// cerrado, ver sección 9.14) — el usuario no debería ver un rechazo por un detalle de
	// certificación que el sistema puede resolver solo.
	if interpreted.IsTestSetClosed() {
		_ = s.numbering.ClearTestSetID(ctx, d.NumberingRangeID) // best-effort, mismo criterio que ReleaseIfCurrent
		return s.sendSyncAndUpdate(ctx, d, cert, key, iss, xmlBytes, number, now, kind)
	}

	status := StatusAccepted
	if interpreted.HasRejections() || !interpreted.IsValid {
		status = StatusRejected
	}
	return s.finish(ctx, d, status, result.ZipKey, interpreted.StatusCode, interpreted.StatusDescription, interpreted.StatusMessage, string(interpreted.ApplicationResponseXML))
}

// sendSyncAndUpdate envía el documento por SendBillSync — un solo documento, síncrono, sin
// ZipKey ni sondeo posterior: la DIAN responde con el resultado final en la misma llamada.
// Confirmado contra la DIAN real (cofacture/soap/realsend_sync_test.go) que esto sigue
// disponible en habilitación incluso con el set de pruebas oficial ya cerrado — ver
// docs/apidian-architecture.md sección 9.14.
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
		return s.finish(ctx, d, StatusSendError, "", resp.StatusCode, "", fmt.Sprintf("interpretar respuesta: %v", err), "")
	}

	status := StatusAccepted
	if interpreted.HasRejections() || !interpreted.IsValid {
		status = StatusRejected
	}
	return s.finish(ctx, d, status, "", interpreted.StatusCode, interpreted.StatusDescription, interpreted.StatusMessage, string(interpreted.ApplicationResponseXML))
}

func (s *Service) markSendError(ctx context.Context, d *Document, sendErr error) (*Document, error) {
	return s.finish(ctx, d, StatusSendError, "", "", "", sendErr.Error(), "")
}

func (s *Service) finish(ctx context.Context, d *Document, status Status, trackID, statusCode, statusDescription, statusMessage, applicationResponseXML string) (*Document, error) {
	if err := s.repo.UpdateDianStatus(ctx, d.ID, status, trackID, statusCode, statusDescription, statusMessage, applicationResponseXML); err != nil {
		return nil, err
	}
	d.Status = status
	d.DianTrackID = trackID
	d.DianStatusCode = statusCode
	d.DianStatusDescription = statusDescription
	d.DianStatusMessage = statusMessage
	d.ApplicationResponseXML = applicationResponseXML

	// El número nunca quedó realmente registrado ante la DIAN (rechazado) ni siquiera se logró
	// transmitir (send_error) — se intenta devolver para que el siguiente intento lo reclame de
	// nuevo en vez de avanzar y dejar un hueco permanente (sección 9.33). Best-effort a
	// propósito: si falla o si ya no es el número vigente del rango (alguien más reclamó otro
	// desde entonces), el documento ya quedó con el estado correcto de todas formas — no se
	// propaga el error de un paso que es una optimización, no una garantía de corrección.
	if status == StatusRejected || status == StatusSendError {
		_ = s.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, d.Number)
	}
	return d, nil
}

// ── Validaciones ───────────────────────────────────────────────────────────────────────────

// validateBase exige paymentMeans no vacío porque cac:PaymentMeans es obligatorio (1..N) para
// Invoice/CreditNote/DebitNote según el Anexo Técnico — confirmado real: 3 facturas
// rechazadas por la DIAN ("errores en campos mandatorios") por mandarlo vacío (ver
// docs/apidian-architecture.md sección 9.30).
//
// También valida cada código de payment_means y de customer.liability_codes contra su
// catálogo en Postgres — payment_terms/payment_methods NUNCA tuvieron FK posible aquí porque
// viven en JSONB (payment_means), y liability_codes tampoco porque es un TEXT[] (Postgres no
// soporta FK contra cada elemento de un array) — ver auditoría de catálogos huérfanos,
// docs/apidian-architecture.md. Es un método de Service (no función libre) justo por esto:
// necesita s.catalogs para consultar la base de datos. lines[].UnitCode NO se valida así —
// ver el comentario de CatalogPort en ports.go.
func (s *Service) validateBase(ctx context.Context, issuerID, rangeID uuid.UUID, lines []LineInput, customer domain.Party, paymentMeans []domain.PaymentMean) error {
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
	if len(paymentMeans) == 0 {
		return ErrMissingPaymentMeans
	}
	for _, pm := range paymentMeans {
		ok, err := s.catalogs.IsValidPaymentTerm(ctx, pm.Code)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %q", ErrInvalidPaymentTerm, pm.Code)
		}
		ok, err = s.catalogs.IsValidPaymentMethod(ctx, pm.PaymentMethodCode)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %q", ErrInvalidPaymentMethod, pm.PaymentMethodCode)
		}
	}
	for _, code := range customer.LiabilityCodes {
		ok, err := s.catalogs.IsValidLiabilityCode(ctx, code)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %q", ErrInvalidLiabilityCode, code)
		}
	}
	return nil
}

func (s *Service) validateNoteRequest(ctx context.Context, req IssueNoteRequest) error {
	if err := s.validateBase(ctx, req.IssuerID, req.NumberingRangeID, req.Lines, req.Customer, req.PaymentMeans); err != nil {
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
	// TaxRegimeCode es *string en issuers.Issuer (FK opcional, ver model.go) pero string
	// plano en domain.Party — domain.Party no distingue "nunca se mandó" de "vacío", el
	// builder simplemente omite el atributo listName si queda vacío.
	var taxRegimeCode string
	if iss.TaxRegimeCode != nil {
		taxRegimeCode = *iss.TaxRegimeCode
	}
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
		LiabilityCodes:              iss.LiabilityCodes,
		TaxRegimeCode:               taxRegimeCode,
		IndustryClassificationCodes: iss.IndustryClassificationCodes,
		TaxSchemeCode:               iss.TaxSchemeCode,
		TaxSchemeName:               iss.TaxSchemeName,
		Phone:                       iss.Phone,
		Email:                       iss.Email,
		MerchantRegistrationNumber:  iss.MerchantRegistrationNumber,
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
//
// CountryCode/CountryName se agregaron tras un rechazo real (StatusCode 99, "errores en
// campos mandatorios", ver docs/apidian-architecture.md sección 9.44): partyFromIssuer ya
// hardcodea "CO"/"Colombia" para el emisor, pero nunca se replicó este default para el
// cliente — invisible mientras ningún cliente de prueba tuviera dirección (cac:Country solo
// se renderiza dentro de cac:RegistrationAddress, que a su vez solo aparece si hay dirección),
// hasta que un cliente real con dirección completa lo expuso. builder.appendAddressFields
// solo agrega cac:Country si CountryCode no está vacío — sin este default, la dirección del
// cliente quedaba sin país, que la DIAN exige como mandatorio.
func applyCustomerDefaults(p *domain.Party) {
	if p.EntityTypeCode == "" {
		p.EntityTypeCode = "2"
	}
	if p.TaxSchemeCode == "" {
		p.TaxSchemeCode = "ZZ"
	}
	if len(p.LiabilityCodes) == 0 {
		p.LiabilityCodes = []string{"R-99-PN"}
	}
	if p.Address.CountryCode == "" {
		p.Address.CountryCode = "CO"
		p.Address.CountryName = "Colombia"
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

// aggregateTaxes agrupa los impuestos de todas las líneas por (TypeCode, Percent) para
// construir cac:TaxTotal de cabecera — la DIAN exige que la base imponible total coincida con
// la suma de las bases de las líneas (regla FAU04, ver comentario en realsend_test.go).
//
// La clave incluye Percent, no solo TypeCode: desde que linesFromInput defaultea "sin
// impuesto" a IVA al 0% (sección 9.46), una factura puede tener líneas con IVA al 19% Y líneas
// con IVA al 0% — agrupar solo por TypeCode las fusionaría en un único cac:TaxSubtotal con un
// Percent equivocado (la base de las líneas al 0% terminaría reportada como si fuera al 19%).
func aggregateTaxes(lines []domain.Line) []domain.Tax {
	type key struct {
		typeCode string
		percent  float64
	}
	type agg struct {
		typeName           string
		taxableAmountCents int64
		taxAmountCents     int64
	}
	byKey := make(map[key]*agg)
	var order []key

	for _, l := range lines {
		for _, t := range l.Taxes {
			k := key{typeCode: t.TypeCode, percent: t.Percent}
			a, ok := byKey[k]
			if !ok {
				a = &agg{typeName: t.TypeName}
				byKey[k] = a
				order = append(order, k)
			}
			a.taxableAmountCents += t.TaxableAmountCents
			a.taxAmountCents += t.TaxAmountCents
		}
	}

	taxes := make([]domain.Tax, 0, len(order))
	for _, k := range order {
		a := byKey[k]
		taxes = append(taxes, domain.Tax{
			TaxableAmountCents: a.taxableAmountCents,
			TaxAmountCents:     a.taxAmountCents,
			Percent:            k.percent,
			TypeCode:           k.typeCode,
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
