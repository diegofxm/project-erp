package application

import (
	"context"
	"fmt"
	"math"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/domain"
)

// CreateDraftUseCase crea borradores para cualquier tipo de documento DIAN.
// No reclama número ni firma — eso ocurre solo en ConfirmUseCase.
type CreateDraftUseCase struct {
	documents domain.DocumentRepository
	numbering domain.NumberingRepository
	companies domain.CompanyPort
	catalogs  domain.CatalogPort
}

// NewCreateDraftUseCase crea el caso de uso.
func NewCreateDraftUseCase(
	documents domain.DocumentRepository,
	numbering domain.NumberingRepository,
	companies domain.CompanyPort,
	catalogs domain.CatalogPort,
) *CreateDraftUseCase {
	return &CreateDraftUseCase{
		documents: documents,
		numbering: numbering,
		companies: companies,
		catalogs:  catalogs,
	}
}

// InvoiceDraftRequest es el payload de una Factura Electrónica de Venta.
type InvoiceDraftRequest struct {
	CompanyID        uuid.UUID
	NumberingRangeID uuid.UUID
	Customer         cofdom.Party
	Lines            []cofdom.Line
	PaymentMeans     []cofdom.PaymentMean
	Note             string
	CurrencyCode     string
	CustomerID       *uuid.UUID
}

// NoteDraftRequest es el payload de NC o ND.
type NoteDraftRequest struct {
	CompanyID           uuid.UUID
	NumberingRangeID    uuid.UUID
	Customer            cofdom.Party
	Lines               []cofdom.Line
	PaymentMeans        []cofdom.PaymentMean
	Note                string
	CurrencyCode        string
	CustomerID          *uuid.UUID
	BillingReference    domain.BillingReferenceInput
	DiscrepancyResponse *domain.DiscrepancyResponseInput
	CreditNoteTypeCode  string // solo NC
}

// SupportDocumentDraftRequest es el payload de un Documento Soporte.
type SupportDocumentDraftRequest struct {
	CompanyID         uuid.UUID
	NumberingRangeID  uuid.UUID
	Supplier          cofdom.Party
	Lines             []cofdom.Line
	PaymentMeans      []cofdom.PaymentMean
	Note              string
	CurrencyCode      string
	OperationTypeCode string // "10" Residente, "11" No Residente
	WithholdingTaxes  []cofdom.Tax
	SupplierID        *uuid.UUID
}

// AdjustmentNoteDraftRequest es el payload de una Nota de Ajuste al Documento Soporte.
type AdjustmentNoteDraftRequest struct {
	CompanyID           uuid.UUID
	NumberingRangeID    uuid.UUID
	Supplier            cofdom.Party
	Lines               []cofdom.Line
	PaymentMeans        []cofdom.PaymentMean
	Note                string
	CurrencyCode        string
	OperationTypeCode   string
	WithholdingTaxes    []cofdom.Tax
	SupplierID          *uuid.UUID
	BillingReference    domain.BillingReferenceInput
	DiscrepancyResponse *domain.DiscrepancyResponseInput
}

const (
	dianFE = "01"
	dianNC = "91"
	dianND = "92"
	dianDS = "05"
	dianNA = "95"
)

func (uc *CreateDraftUseCase) CreateInvoiceDraft(ctx context.Context, req InvoiceDraftRequest) (*domain.Document, error) {
	if err := uc.validateBase(ctx, req.CompanyID, req.NumberingRangeID, dianFE, req.Lines, req.Customer, req.PaymentMeans); err != nil {
		return nil, err
	}
	applyCustomerDefaults(&req.Customer)
	if err := uc.resolveTaxTypeName(ctx, req.Lines); err != nil {
		return nil, err
	}
	d := domain.Document{
		CompanyID:            req.CompanyID,
		NumberingRangeID:     req.NumberingRangeID,
		DianDocumentTypeCode: dianFE,
		CurrencyCode:         defaultCurrency(req.CurrencyCode),
		Note:                 req.Note,
		Customer:             req.Customer,
		CustomerID:           req.CustomerID,
		Lines:                req.Lines,
		PaymentMeans:         req.PaymentMeans,
		Totals:               computeTotals(req.Lines),
		Status:               domain.StatusDraft,
	}
	return uc.documents.Create(ctx, d)
}

func (uc *CreateDraftUseCase) CreateCreditNoteDraft(ctx context.Context, req NoteDraftRequest) (*domain.Document, error) {
	if err := uc.validateNote(ctx, req.CompanyID, req.NumberingRangeID, dianNC, req.Lines, req.Customer, req.PaymentMeans, req.BillingReference.CUFE); err != nil {
		return nil, err
	}
	applyCustomerDefaults(&req.Customer)
	if err := uc.resolveTaxTypeName(ctx, req.Lines); err != nil {
		return nil, err
	}
	d := noteDraft(req, dianNC)
	return uc.documents.Create(ctx, d)
}

func (uc *CreateDraftUseCase) CreateDebitNoteDraft(ctx context.Context, req NoteDraftRequest) (*domain.Document, error) {
	if err := uc.validateNote(ctx, req.CompanyID, req.NumberingRangeID, dianND, req.Lines, req.Customer, req.PaymentMeans, req.BillingReference.CUFE); err != nil {
		return nil, err
	}
	applyCustomerDefaults(&req.Customer)
	if err := uc.resolveTaxTypeName(ctx, req.Lines); err != nil {
		return nil, err
	}
	d := noteDraft(req, dianND)
	return uc.documents.Create(ctx, d)
}

func (uc *CreateDraftUseCase) CreateSupportDocumentDraft(ctx context.Context, req SupportDocumentDraftRequest) (*domain.Document, error) {
	if err := uc.validateSupport(ctx, req.CompanyID, req.NumberingRangeID, req.Lines, req.Supplier, req.PaymentMeans, req.OperationTypeCode); err != nil {
		return nil, err
	}
	applySupplierDefaults(&req.Supplier)
	if err := uc.resolveTaxTypeName(ctx, req.Lines); err != nil {
		return nil, err
	}
	supplier := req.Supplier
	d := domain.Document{
		CompanyID:            req.CompanyID,
		NumberingRangeID:     req.NumberingRangeID,
		DianDocumentTypeCode: dianDS,
		CurrencyCode:         defaultCurrency(req.CurrencyCode),
		Note:                 req.Note,
		Lines:                req.Lines,
		PaymentMeans:         req.PaymentMeans,
		Totals:               computeTotalsDS(req.Lines, req.WithholdingTaxes),
		Supplier:             &supplier,
		SupplierID:           req.SupplierID,
		OperationTypeCode:    req.OperationTypeCode,
		WithholdingTaxes:     req.WithholdingTaxes,
		Status:               domain.StatusDraft,
	}
	return uc.documents.Create(ctx, d)
}

func (uc *CreateDraftUseCase) CreateAdjustmentNoteDraft(ctx context.Context, req AdjustmentNoteDraftRequest) (*domain.Document, error) {
	if err := uc.validateSupport(ctx, req.CompanyID, req.NumberingRangeID, req.Lines, req.Supplier, req.PaymentMeans, req.OperationTypeCode); err != nil {
		return nil, err
	}
	if req.BillingReference.CUFE == "" {
		return nil, domain.ErrMissingBillingReference
	}
	applySupplierDefaults(&req.Supplier)
	if err := uc.resolveTaxTypeName(ctx, req.Lines); err != nil {
		return nil, err
	}
	supplier := req.Supplier
	billingRef := req.BillingReference
	d := domain.Document{
		CompanyID:            req.CompanyID,
		NumberingRangeID:     req.NumberingRangeID,
		DianDocumentTypeCode: dianNA,
		CurrencyCode:         defaultCurrency(req.CurrencyCode),
		Note:                 req.Note,
		Lines:                req.Lines,
		PaymentMeans:         req.PaymentMeans,
		Totals:               computeTotalsDS(req.Lines, req.WithholdingTaxes),
		Supplier:             &supplier,
		SupplierID:           req.SupplierID,
		OperationTypeCode:    req.OperationTypeCode,
		WithholdingTaxes:     req.WithholdingTaxes,
		BillingReference:     &billingRef,
		DiscrepancyResponse:  req.DiscrepancyResponse,
		Status:               domain.StatusDraft,
	}
	return uc.documents.Create(ctx, d)
}

func (uc *CreateDraftUseCase) DeleteDraft(ctx context.Context, companyID, id uuid.UUID) error {
	d, err := uc.documents.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d.CompanyID != companyID {
		return domain.ErrDocumentNotFound
	}
	if d.Status != domain.StatusDraft {
		return domain.ErrDocumentNotDraft
	}
	return uc.documents.Delete(ctx, id)
}

// ── validaciones ──────────────────────────────────────────────────────────────────────────

func (uc *CreateDraftUseCase) validateBase(ctx context.Context, companyID, rangeID uuid.UUID, dianDocType string, lines []cofdom.Line, customer cofdom.Party, paymentMeans []cofdom.PaymentMean) error {
	if companyID == uuid.Nil {
		return domain.ErrMissingCompany
	}
	if rangeID == uuid.Nil {
		return domain.ErrMissingNumberingRange
	}
	if len(lines) == 0 {
		return domain.ErrEmptyLines
	}
	if customer.Identification.Number == "" {
		return domain.ErrMissingCustomer
	}
	if len(paymentMeans) == 0 {
		return domain.ErrMissingPaymentMeans
	}
	if err := uc.validatePaymentMeans(ctx, paymentMeans); err != nil {
		return err
	}
	nr, err := uc.numbering.GetByID(ctx, rangeID)
	if err != nil {
		return err
	}
	if nr.CompanyID != companyID {
		return domain.ErrRangeCompanyMismatch
	}
	if nr.DianDocumentTypeCode != dianDocType {
		return domain.ErrWrongDocumentType
	}
	return nil
}

func (uc *CreateDraftUseCase) validateNote(ctx context.Context, companyID, rangeID uuid.UUID, dianDocType string, lines []cofdom.Line, customer cofdom.Party, paymentMeans []cofdom.PaymentMean, cufe string) error {
	if err := uc.validateBase(ctx, companyID, rangeID, dianDocType, lines, customer, paymentMeans); err != nil {
		return err
	}
	if cufe == "" {
		return domain.ErrMissingBillingReference
	}
	return nil
}

func (uc *CreateDraftUseCase) validateSupport(ctx context.Context, companyID, rangeID uuid.UUID, lines []cofdom.Line, supplier cofdom.Party, paymentMeans []cofdom.PaymentMean, operationTypeCode string) error {
	if companyID == uuid.Nil {
		return domain.ErrMissingCompany
	}
	if rangeID == uuid.Nil {
		return domain.ErrMissingNumberingRange
	}
	if len(lines) == 0 {
		return domain.ErrEmptyLines
	}
	if supplier.Identification.Number == "" {
		return domain.ErrMissingSupplier
	}
	if operationTypeCode != "10" && operationTypeCode != "11" {
		return domain.ErrInvalidOperationTypeCode
	}
	if len(paymentMeans) == 0 {
		return domain.ErrMissingPaymentMeans
	}
	if err := uc.validatePaymentMeans(ctx, paymentMeans); err != nil {
		return err
	}
	nr, err := uc.numbering.GetByID(ctx, rangeID)
	if err != nil {
		return err
	}
	if nr.CompanyID != companyID {
		return domain.ErrRangeCompanyMismatch
	}
	return nil
}

func (uc *CreateDraftUseCase) validatePaymentMeans(ctx context.Context, pms []cofdom.PaymentMean) error {
	for _, pm := range pms {
		ok, err := uc.catalogs.IsValidPaymentTerm(ctx, pm.Code)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %q", domain.ErrInvalidPaymentTerm, pm.Code)
		}
		ok, err = uc.catalogs.IsValidPaymentMethod(ctx, pm.PaymentMethodCode)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %q", domain.ErrInvalidPaymentMethod, pm.PaymentMethodCode)
		}
	}
	return nil
}

func (uc *CreateDraftUseCase) resolveTaxTypeName(ctx context.Context, lines []cofdom.Line) error {
	for i := range lines {
		for j := range lines[i].Taxes {
			if lines[i].Taxes[j].TypeName == "" && lines[i].Taxes[j].TypeCode != "" {
				name, found, err := uc.catalogs.GetTaxTypeName(ctx, lines[i].Taxes[j].TypeCode)
				if err != nil {
					return err
				}
				if found {
					lines[i].Taxes[j].TypeName = name
				}
			}
		}
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────────────────

func noteDraft(req NoteDraftRequest, dianDocType string) domain.Document {
	billingRef := req.BillingReference
	d := domain.Document{
		CompanyID:            req.CompanyID,
		NumberingRangeID:     req.NumberingRangeID,
		DianDocumentTypeCode: dianDocType,
		CurrencyCode:         defaultCurrency(req.CurrencyCode),
		Note:                 req.Note,
		Customer:             req.Customer,
		CustomerID:           req.CustomerID,
		Lines:                req.Lines,
		PaymentMeans:         req.PaymentMeans,
		Totals:               computeTotals(req.Lines),
		BillingReference:     &billingRef,
		DiscrepancyResponse:  req.DiscrepancyResponse,
		NoteTypeCode:         req.CreditNoteTypeCode,
		Status:               domain.StatusDraft,
	}
	return d
}

func computeTotals(lines []cofdom.Line) cofdom.Totals {
	var lineExt, taxAmount int64
	for _, l := range lines {
		lineExt += l.LineExtensionCents
		for _, t := range l.Taxes {
			taxAmount += t.TaxAmountCents
		}
	}
	taxInclusive := lineExt + taxAmount
	return cofdom.Totals{
		LineExtensionCents: lineExt,
		TaxExclusiveCents:  lineExt,
		TaxInclusiveCents:  taxInclusive,
		PayableCents:       taxInclusive,
	}
}

func computeTotalsDS(lines []cofdom.Line, withholdingTaxes []cofdom.Tax) cofdom.Totals {
	var lineExt, lineTaxAmount int64
	for _, l := range lines {
		lineExt += l.LineExtensionCents
		for _, t := range l.Taxes {
			lineTaxAmount += t.TaxAmountCents
		}
	}
	taxInclusive := lineExt + lineTaxAmount
	return cofdom.Totals{
		LineExtensionCents: lineExt,
		TaxExclusiveCents:  lineExt,
		TaxInclusiveCents:  taxInclusive,
		PayableCents:       taxInclusive,
	}
}

// ── cálculo de líneas desde input simplificado ────────────────────────────────────────────

// LineInputData es la forma de entrada de una línea — el servidor calcula
// line_extension_cents y taxes[] a partir de quantity/unit_price_cents/tax_percent.
// Mismo contrato que legacy linesFromInput (_legacy/apidian/internal/edocuments/documents/lines.go).
type LineInputData struct {
	Description    string
	Quantity       float64
	UnitCode       string
	UnitPriceCents int64
	ItemCode       string
	ItemTypeCode   string // vacío → "999" (estándar propio)
	TaxTypeCode    string // vacío → "01" (IVA) con TaxPercent=0
	TaxPercent     float64
}

// itemStandards es el catálogo fijo del Anexo Técnico DIAN tabla 13.3.5.
var itemStandards = map[string]struct{ name, agencyID string }{
	"001": {"UNSPSC", "113"},
	"010": {"GTIN", "6"},
	"020": {"Partida Arancelaria", "9"},
	"999": {"Estándar de adopción del contribuyente", "ZZ"},
}

// LinesFromInput computa []cofdom.Line completas desde []LineInputData:
// - line_extension_cents = round(quantity × unit_price_cents)
// - tax_amount_cents = round(line_extension × tax_percent / 100)
// - Si TaxTypeCode vacío → "01" (IVA) al 0%
// - Resuelve TypeName desde catálogo
func (uc *CreateDraftUseCase) LinesFromInput(ctx context.Context, inputs []LineInputData) ([]cofdom.Line, error) {
	lines := make([]cofdom.Line, len(inputs))
	for i, in := range inputs {
		lineExt := roundCents(in.Quantity * float64(in.UnitPriceCents))
		itemTypeCode := in.ItemTypeCode
		if itemTypeCode == "" {
			itemTypeCode = "999"
		}
		std, ok := itemStandards[itemTypeCode]
		if !ok {
			return nil, fmt.Errorf("%w: %q", domain.ErrInvalidItemStandardCode, itemTypeCode)
		}
		itemCode := in.ItemCode
		if itemCode == "" {
			// DIAN exige un valor no vacío en StandardItemIdentification/ID.
			// Para "999" (estándar propio) usamos el número de línea como código interno.
			itemCode = fmt.Sprintf("%d", i+1)
		}
		line := cofdom.Line{
			Description:        in.Description,
			Quantity:           in.Quantity,
			UnitCode:           in.UnitCode,
			LineExtensionCents: lineExt,
			UnitPriceCents:     in.UnitPriceCents,
			ItemCode:           itemCode,
			ItemTypeCode:       itemTypeCode,
			ItemTypeName:       std.name,
			ItemTypeAgencyID:   std.agencyID,
		}
		taxTypeCode := in.TaxTypeCode
		taxPercent := in.TaxPercent
		if taxTypeCode == "" {
			taxTypeCode = "01"
			taxPercent = 0
		}
		name, found, err := uc.catalogs.GetTaxTypeName(ctx, taxTypeCode)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("%w: %q", domain.ErrInvalidTaxTypeCode, taxTypeCode)
		}
		taxAmt := roundCents(float64(lineExt) * taxPercent / 100)
		line.Taxes = []cofdom.Tax{{
			TaxableAmountCents: lineExt,
			TaxAmountCents:     taxAmt,
			Percent:            taxPercent,
			TypeCode:           taxTypeCode,
			TypeName:           name,
		}}
		lines[i] = line
	}
	return lines, nil
}

func roundCents(v float64) int64 {
	return int64(math.Round(v))
}

func applyCustomerDefaults(p *cofdom.Party) {
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

func applySupplierDefaults(p *cofdom.Party) {
	if p.EntityTypeCode == "" {
		p.EntityTypeCode = "1"
	}
	if p.TaxSchemeCode == "" {
		p.TaxSchemeCode = "ZZ"
		p.TaxSchemeName = "No aplica"
	}
	if len(p.LiabilityCodes) == 0 {
		p.LiabilityCodes = []string{"O-47"}
	}
	if p.TaxRegimeCode == "" {
		p.TaxRegimeCode = "49"
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
