package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/application"
	"github.com/diegofxm/erp/internal/electronic/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

// AuditLogger es la interfaz mínima para registrar eventos de auditoría.
// La implementa audit/application.UseCase sin requerir importar ese paquete.
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

// Handler agrupa todos los handlers del módulo electronic.
type Handler struct {
	createDraft application.CreateDraftUseCase
	confirm     application.ConfirmUseCase
	get         application.GetDocumentUseCase
	list        application.ListDocumentsUseCase
	numbering   application.ManageNumberingUseCase
	pdf         *application.GetDocumentPDFUseCase
	fromSale    *application.CreateFromSaleUseCase
	dianRanges  *application.GetDianRangesUseCase
	audit       AuditLogger
}

func NewHandler(
	createDraft *application.CreateDraftUseCase,
	confirm *application.ConfirmUseCase,
	get *application.GetDocumentUseCase,
	list *application.ListDocumentsUseCase,
	numbering *application.ManageNumberingUseCase,
	pdf *application.GetDocumentPDFUseCase,
	fromSale *application.CreateFromSaleUseCase,
	dianRanges *application.GetDianRangesUseCase,
	audit AuditLogger,
) *Handler {
	return &Handler{
		createDraft: *createDraft,
		confirm:     *confirm,
		get:         *get,
		list:        *list,
		numbering:   *numbering,
		pdf:         pdf,
		fromSale:    fromSale,
		dianRanges:  dianRanges,
		audit:       audit,
	}
}

// ── Documentos ────────────────────────────────────────────────────────────────────────────

func (h *Handler) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	docs, err := h.list.List(r.Context(), companyID, domain.ListFilter{Limit: 50})
	if err != nil {
		writeErr(w, err)
		return
	}
	if docs == nil {
		docs = []*domain.Document{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"documents": docs,
		"count":     len(docs),
	})
}

func (h *Handler) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	doc, err := h.get.Get(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) handleConfirmDocument(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	doc, err := h.confirm.Confirm(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.confirmed", doc)
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) handleGetDocumentPDF(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	format := r.URL.Query().Get("format")
	pdfBytes, err := h.pdf.GetPDF(r.Context(), companyID, id, format)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="documento.pdf"`)
	_, _ = w.Write(pdfBytes)
}

func (h *Handler) handleDeleteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if err := h.createDraft.DeleteDraft(r.Context(), companyID, id); err != nil {
		writeErr(w, err)
		return
	}
	if h.audit != nil {
		uid := tenant.GetUserID(r.Context())
		var userID *uuid.UUID
		if uid != uuid.Nil {
			userID = &uid
		}
		h.audit.Log(r.Context(), companyID, userID, "document.deleted", "document", &id, nil)
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── DTOs de entrada snake_case ────────────────────────────────────────────────────────────
// cofacture/domain.* no tiene json tags — estos DTOs son el contrato HTTP público.

type identificationInputDTO struct {
	Number           string `json:"number"`
	TypeCode         string `json:"type_code"`
	VerificationCode string `json:"verification_code,omitempty"`
}

type addressInputDTO struct {
	Line        string `json:"line,omitempty"`
	CityCode    string `json:"city_code,omitempty"`
	CityName    string `json:"city_name,omitempty"`
	PostalZone  string `json:"postal_zone,omitempty"`
	StateCode   string `json:"state_code,omitempty"`
	StateName   string `json:"state_name,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	CountryName string `json:"country_name,omitempty"`
}

type partyInputDTO struct {
	EntityTypeCode             string                 `json:"entity_type_code,omitempty"`
	Identification             identificationInputDTO `json:"identification"`
	Name                       string                 `json:"name"`
	Address                    *addressInputDTO       `json:"address,omitempty"`
	TaxSchemeCode              string                 `json:"tax_scheme_code,omitempty"`
	TaxSchemeName              string                 `json:"tax_scheme_name,omitempty"`
	LiabilityCodes             []string               `json:"liability_codes,omitempty"`
	TaxRegimeCode              string                 `json:"tax_regime_code,omitempty"`
	Phone                      string                 `json:"phone,omitempty"`
	Email                      string                 `json:"email,omitempty"`
	MerchantRegistrationNumber *string                `json:"merchant_registration_number,omitempty"`
}

func (p partyInputDTO) toCofdom() cofdom.Party {
	party := cofdom.Party{
		EntityTypeCode: p.EntityTypeCode,
		Identification: cofdom.Identification{
			Number:           p.Identification.Number,
			TypeCode:         p.Identification.TypeCode,
			VerificationCode: p.Identification.VerificationCode,
		},
		Name:                       p.Name,
		TaxSchemeCode:              p.TaxSchemeCode,
		TaxSchemeName:              p.TaxSchemeName,
		LiabilityCodes:             p.LiabilityCodes,
		TaxRegimeCode:              p.TaxRegimeCode,
		Phone:                      p.Phone,
		Email:                      p.Email,
		MerchantRegistrationNumber: p.MerchantRegistrationNumber,
	}
	if p.Address != nil {
		party.Address = cofdom.Address{
			Line:        p.Address.Line,
			CityCode:    p.Address.CityCode,
			CityName:    p.Address.CityName,
			PostalZone:  p.Address.PostalZone,
			StateCode:   p.Address.StateCode,
			StateName:   p.Address.StateName,
			CountryCode: p.Address.CountryCode,
			CountryName: p.Address.CountryName,
		}
	}
	return party
}

type lineInputDTO struct {
	Description    string  `json:"description"`
	Quantity       float64 `json:"quantity"`
	UnitCode       string  `json:"unit_code"`
	UnitPriceCents int64   `json:"unit_price_cents"`
	ItemCode       string  `json:"item_code,omitempty"`
	ItemTypeCode   string  `json:"item_type_code,omitempty"`
	TaxTypeCode    string  `json:"tax_type_code,omitempty"`
	TaxPercent     float64 `json:"tax_percent,omitempty"`
}

func (l lineInputDTO) toAppInput() application.LineInputData {
	return application.LineInputData{
		Description:    l.Description,
		Quantity:       l.Quantity,
		UnitCode:       l.UnitCode,
		UnitPriceCents: l.UnitPriceCents,
		ItemCode:       l.ItemCode,
		ItemTypeCode:   l.ItemTypeCode,
		TaxTypeCode:    l.TaxTypeCode,
		TaxPercent:     l.TaxPercent,
	}
}

func linesInputToApp(dtos []lineInputDTO) []application.LineInputData {
	out := make([]application.LineInputData, len(dtos))
	for i, d := range dtos {
		out[i] = d.toAppInput()
	}
	return out
}

type paymentMeanInputDTO struct {
	Code              string `json:"code"`
	PaymentMethodCode string `json:"payment_method_code"`
	DueDate           string `json:"due_date,omitempty"`
	PaymentReference  string `json:"payment_reference,omitempty"`
}

func (pm paymentMeanInputDTO) toCofdom() cofdom.PaymentMean {
	return cofdom.PaymentMean{
		Code:              pm.Code,
		PaymentMethodCode: pm.PaymentMethodCode,
		DueDate:           pm.DueDate,
		PaymentReference:  pm.PaymentReference,
	}
}

func paymentMeansInputToCofdom(pms []paymentMeanInputDTO) []cofdom.PaymentMean {
	out := make([]cofdom.PaymentMean, len(pms))
	for i, pm := range pms {
		out[i] = pm.toCofdom()
	}
	return out
}

type taxInputDTO struct {
	TaxableAmountCents int64   `json:"taxable_amount_cents"`
	TaxAmountCents     int64   `json:"tax_amount_cents"`
	Percent            float64 `json:"percent"`
	TypeCode           string  `json:"type_code"`
	TypeName           string  `json:"type_name,omitempty"`
}

func (t taxInputDTO) toCofdom() cofdom.Tax {
	return cofdom.Tax{
		TaxableAmountCents: t.TaxableAmountCents,
		TaxAmountCents:     t.TaxAmountCents,
		Percent:            t.Percent,
		TypeCode:           t.TypeCode,
		TypeName:           t.TypeName,
	}
}

func taxesInputToCofdom(taxes []taxInputDTO) []cofdom.Tax {
	out := make([]cofdom.Tax, len(taxes))
	for i, t := range taxes {
		out[i] = t.toCofdom()
	}
	return out
}

// ── Crear borradores ──────────────────────────────────────────────────────────────────────

type invoiceDraftBody struct {
	NumberingRangeID uuid.UUID             `json:"numbering_range_id"`
	Customer         partyInputDTO         `json:"customer"`
	Lines            []lineInputDTO        `json:"lines"`
	PaymentMeans     []paymentMeanInputDTO `json:"payment_means"`
	Note             string                `json:"note"`
	CurrencyCode     string                `json:"currency_code"`
	CustomerID       *uuid.UUID            `json:"customer_id"`
}

func (h *Handler) handleCreateInvoiceDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body invoiceDraftBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lines, err := h.createDraft.LinesFromInput(r.Context(), linesInputToApp(body.Lines))
	if err != nil {
		writeErr(w, err)
		return
	}
	doc, err := h.createDraft.CreateInvoiceDraft(r.Context(), application.InvoiceDraftRequest{
		CompanyID:        companyID,
		NumberingRangeID: body.NumberingRangeID,
		Customer:         body.Customer.toCofdom(),
		Lines:            lines,
		PaymentMeans:     paymentMeansInputToCofdom(body.PaymentMeans),
		Note:             body.Note,
		CurrencyCode:     body.CurrencyCode,
		CustomerID:       body.CustomerID,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.created", doc)
	writeJSON(w, http.StatusCreated, doc)
}

type noteDraftBody struct {
	NumberingRangeID    uuid.UUID                        `json:"numbering_range_id"`
	Customer            partyInputDTO                    `json:"customer"`
	Lines               []lineInputDTO                   `json:"lines"`
	PaymentMeans        []paymentMeanInputDTO            `json:"payment_means"`
	Note                string                           `json:"note"`
	CurrencyCode        string                           `json:"currency_code"`
	CustomerID          *uuid.UUID                       `json:"customer_id"`
	BillingReference    domain.BillingReferenceInput     `json:"billing_reference"`
	DiscrepancyResponse *domain.DiscrepancyResponseInput `json:"discrepancy_response"`
	CreditNoteTypeCode  string                           `json:"credit_note_type_code"`
}

func (h *Handler) handleCreateCreditNoteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body noteDraftBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lines, err := h.createDraft.LinesFromInput(r.Context(), linesInputToApp(body.Lines))
	if err != nil {
		writeErr(w, err)
		return
	}
	doc, err := h.createDraft.CreateCreditNoteDraft(r.Context(), application.NoteDraftRequest{
		CompanyID:           companyID,
		NumberingRangeID:    body.NumberingRangeID,
		Customer:            body.Customer.toCofdom(),
		Lines:               lines,
		PaymentMeans:        paymentMeansInputToCofdom(body.PaymentMeans),
		Note:                body.Note,
		CurrencyCode:        body.CurrencyCode,
		CustomerID:          body.CustomerID,
		BillingReference:    body.BillingReference,
		DiscrepancyResponse: body.DiscrepancyResponse,
		CreditNoteTypeCode:  body.CreditNoteTypeCode,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *Handler) handleCreateDebitNoteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body noteDraftBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lines, err := h.createDraft.LinesFromInput(r.Context(), linesInputToApp(body.Lines))
	if err != nil {
		writeErr(w, err)
		return
	}
	doc, err := h.createDraft.CreateDebitNoteDraft(r.Context(), application.NoteDraftRequest{
		CompanyID:           companyID,
		NumberingRangeID:    body.NumberingRangeID,
		Customer:            body.Customer.toCofdom(),
		Lines:               lines,
		PaymentMeans:        paymentMeansInputToCofdom(body.PaymentMeans),
		Note:                body.Note,
		CurrencyCode:        body.CurrencyCode,
		CustomerID:          body.CustomerID,
		BillingReference:    body.BillingReference,
		DiscrepancyResponse: body.DiscrepancyResponse,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

type supportDocBody struct {
	NumberingRangeID  uuid.UUID             `json:"numbering_range_id"`
	Vendor            partyInputDTO         `json:"vendor"`
	Lines             []lineInputDTO        `json:"lines"`
	PaymentMeans      []paymentMeanInputDTO `json:"payment_means"`
	Note              string                `json:"note"`
	CurrencyCode      string                `json:"currency_code"`
	OperationTypeCode string                `json:"operation_type_code"`
	WithholdingTaxes  []taxInputDTO         `json:"withholding_taxes"`
	VendorID          *uuid.UUID            `json:"vendor_id"`
}

func (h *Handler) handleCreateSupportDocDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body supportDocBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lines, err := h.createDraft.LinesFromInput(r.Context(), linesInputToApp(body.Lines))
	if err != nil {
		writeErr(w, err)
		return
	}
	doc, err := h.createDraft.CreateSupportDocumentDraft(r.Context(), application.SupportDocumentDraftRequest{
		CompanyID:         companyID,
		NumberingRangeID:  body.NumberingRangeID,
		Vendor:            body.Vendor.toCofdom(),
		Lines:             lines,
		PaymentMeans:      paymentMeansInputToCofdom(body.PaymentMeans),
		Note:              body.Note,
		CurrencyCode:      body.CurrencyCode,
		OperationTypeCode: body.OperationTypeCode,
		WithholdingTaxes:  taxesInputToCofdom(body.WithholdingTaxes),
		VendorID:          body.VendorID,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

type adjustmentNoteBody struct {
	NumberingRangeID    uuid.UUID                        `json:"numbering_range_id"`
	Vendor              partyInputDTO                    `json:"vendor"`
	Lines               []lineInputDTO                   `json:"lines"`
	PaymentMeans        []paymentMeanInputDTO            `json:"payment_means"`
	Note                string                           `json:"note"`
	CurrencyCode        string                           `json:"currency_code"`
	OperationTypeCode   string                           `json:"operation_type_code"`
	WithholdingTaxes    []taxInputDTO                    `json:"withholding_taxes"`
	VendorID            *uuid.UUID                       `json:"vendor_id"`
	BillingReference    domain.BillingReferenceInput     `json:"billing_reference"`
	DiscrepancyResponse *domain.DiscrepancyResponseInput `json:"discrepancy_response"`
}

func (h *Handler) handleCreateAdjustmentNoteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body adjustmentNoteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lines, err := h.createDraft.LinesFromInput(r.Context(), linesInputToApp(body.Lines))
	if err != nil {
		writeErr(w, err)
		return
	}
	doc, err := h.createDraft.CreateAdjustmentNoteDraft(r.Context(), application.AdjustmentNoteDraftRequest{
		CompanyID:           companyID,
		NumberingRangeID:    body.NumberingRangeID,
		Vendor:              body.Vendor.toCofdom(),
		Lines:               lines,
		PaymentMeans:        paymentMeansInputToCofdom(body.PaymentMeans),
		Note:                body.Note,
		CurrencyCode:        body.CurrencyCode,
		OperationTypeCode:   body.OperationTypeCode,
		WithholdingTaxes:    taxesInputToCofdom(body.WithholdingTaxes),
		VendorID:            body.VendorID,
		BillingReference:    body.BillingReference,
		DiscrepancyResponse: body.DiscrepancyResponse,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

// ── Factura desde venta ───────────────────────────────────────────────────────────────────

type fromSaleBody struct {
	NumberingRangeID uuid.UUID `json:"numbering_range_id"`
}

func (h *Handler) handleCreateInvoiceFromSale(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	saleID, err := uuid.Parse(r.PathValue("sale_id"))
	if err != nil {
		http.Error(w, "sale_id inválido", http.StatusBadRequest)
		return
	}
	var body fromSaleBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	doc, err := h.fromSale.Execute(r.Context(), application.FromSaleRequest{
		CompanyID:        companyID,
		SaleID:           saleID,
		NumberingRangeID: body.NumberingRangeID,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

// ── Rangos de numeración ──────────────────────────────────────────────────────────────────

type numberingRangeBody struct {
	DianDocumentTypeCode string             `json:"dian_document_type_code"`
	Prefix               string             `json:"prefix"`
	ResolutionNumber     string             `json:"resolution_number"`
	ResolutionDate       string             `json:"resolution_date"` // YYYY-MM-DD
	RangeFrom            int64              `json:"range_from"`
	RangeTo              *int64             `json:"range_to"`
	ValidFrom            string             `json:"valid_from"` // YYYY-MM-DD
	ValidTo              string             `json:"valid_to"`
	Environment          domain.Environment `json:"environment"`
	TechnicalKey         string             `json:"technical_key"`
	TestSetID            string             `json:"test_set_id"`
	NextNumber           *int64             `json:"next_number"` // próximo número a emitir; nil = iniciar desde range_from
}

type numberingRangeDTO struct {
	ID                   uuid.UUID          `json:"id"`
	CompanyID            uuid.UUID          `json:"company_id"`
	DianDocumentTypeCode string             `json:"dian_document_type_code"`
	Prefix               string             `json:"prefix"`
	ResolutionNumber     string             `json:"resolution_number"`
	ResolutionDate       string             `json:"resolution_date"`
	RangeFrom            int64              `json:"range_from"`
	RangeTo              *int64             `json:"range_to"`
	CurrentNumber        int64              `json:"current_number"`
	ValidFrom            string             `json:"valid_from"`
	ValidTo              string             `json:"valid_to"`
	Environment          domain.Environment `json:"environment"`
	IsActive             bool               `json:"is_active"`
	Status               string             `json:"status"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func toNumberingRangeDTO(nr *domain.NumberingRange) numberingRangeDTO {
	return numberingRangeDTO{
		ID:                   nr.ID,
		CompanyID:            nr.CompanyID,
		DianDocumentTypeCode: nr.DianDocumentTypeCode,
		Prefix:               nr.Prefix,
		ResolutionNumber:     nr.ResolutionNumber,
		ResolutionDate:       formatDate(nr.ResolutionDate),
		RangeFrom:            nr.RangeFrom,
		RangeTo:              nr.RangeTo,
		CurrentNumber:        nr.CurrentNumber,
		ValidFrom:            formatDate(nr.ValidFrom),
		ValidTo:              formatDate(nr.ValidTo),
		Environment:          nr.Environment,
		IsActive:             nr.IsActive,
		Status:               nr.Status(),
		CreatedAt:            nr.CreatedAt,
		UpdatedAt:            nr.UpdatedAt,
	}
}

func (h *Handler) handleCreateNumberingRange(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body numberingRangeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	nr := domain.NumberingRange{
		CompanyID:            companyID,
		DianDocumentTypeCode: body.DianDocumentTypeCode,
		Prefix:               body.Prefix,
		ResolutionNumber:     body.ResolutionNumber,
		RangeFrom:            body.RangeFrom,
		RangeTo:              body.RangeTo,
		Environment:          body.Environment,
		TechnicalKey:         body.TechnicalKey,
		TestSetID:            body.TestSetID,
		IsActive:             true,
	}
	if body.ResolutionDate != "" {
		if t, err := time.Parse("2006-01-02", body.ResolutionDate); err == nil {
			nr.ResolutionDate = t
		}
	}
	if body.ValidFrom != "" {
		if t, err := time.Parse("2006-01-02", body.ValidFrom); err == nil {
			nr.ValidFrom = t
		}
	}
	if body.ValidTo != "" {
		if t, err := time.Parse("2006-01-02", body.ValidTo); err == nil {
			nr.ValidTo = t
		}
	}
	if nr.RangeFrom <= 0 {
		nr.RangeFrom = 1
	}
	if body.NextNumber != nil && *body.NextNumber >= nr.RangeFrom {
		nr.CurrentNumber = *body.NextNumber - 1
	} else {
		nr.CurrentNumber = nr.RangeFrom - 1
	}
	created, err := h.numbering.Create(r.Context(), nr)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toNumberingRangeDTO(created))
}

func (h *Handler) handleListNumberingRanges(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	docType := r.URL.Query().Get("dian_document_type_code")
	ranges, err := h.numbering.List(r.Context(), companyID, docType)
	if err != nil {
		writeErr(w, err)
		return
	}
	dtos := make([]numberingRangeDTO, len(ranges))
	for i, nr := range ranges {
		dtos[i] = toNumberingRangeDTO(nr)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"numbering_ranges": dtos,
		"count":            len(dtos),
	})
}

func (h *Handler) handleDeactivateRange(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if err := h.numbering.Deactivate(r.Context(), companyID, id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleActivateRange(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	nr, err := h.numbering.Activate(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toNumberingRangeDTO(nr))
}

func (h *Handler) handleGetDianNumberingRanges(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	ranges, err := h.dianRanges.Execute(r.Context(), companyID)
	if err != nil {
		if errors.Is(err, application.ErrCompanyNotReadyForDian) {
			writeErr(w, err)
			return
		}
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ranges": ranges})
}

// ── helpers ───────────────────────────────────────────────────────────────────────────────

// logDoc registra un evento de auditoría para una operación sobre un documento.
func (h *Handler) logDoc(ctx context.Context, companyID uuid.UUID, action string, doc *domain.Document) {
	if h.audit == nil {
		return
	}
	uid := tenant.GetUserID(ctx)
	var userID *uuid.UUID
	if uid != uuid.Nil {
		userID = &uid
	}
	meta := map[string]any{
		"dian_document_type_code": doc.DianDocumentTypeCode,
	}
	if doc.Prefix != "" {
		meta["prefix"] = doc.Prefix
	}
	if doc.Number > 0 {
		meta["number"] = doc.Number
	}
	if doc.Customer.Name != "" {
		meta["customer_name"] = doc.Customer.Name
	} else if doc.Vendor != nil && doc.Vendor.Name != "" {
		meta["vendor_name"] = doc.Vendor.Name
	}
	h.audit.Log(ctx, companyID, userID, action, "document", &doc.ID, meta)
}

func mustTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id := tenant.GetCompanyID(r.Context())
	if id == uuid.Nil {
		http.Error(w, "empresa requerida", http.StatusUnauthorized)
		return uuid.Nil, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrDocumentNotFound),
		errors.Is(err, domain.ErrRangeNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrDocumentNotDraft),
		errors.Is(err, domain.ErrEmptyLines),
		errors.Is(err, domain.ErrMissingCustomer),
		errors.Is(err, domain.ErrMissingPaymentMeans),
		errors.Is(err, domain.ErrMissingBillingReference),
		errors.Is(err, domain.ErrMissingVendor),
		errors.Is(err, domain.ErrRangeCompanyMismatch),
		errors.Is(err, domain.ErrWrongDocumentType),
		errors.Is(err, domain.ErrInvalidPaymentTerm),
		errors.Is(err, domain.ErrInvalidPaymentMethod),
		errors.Is(err, domain.ErrInvalidOperationTypeCode),
		errors.Is(err, domain.ErrRangeExhausted):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrCompanyNotReadyToIssue):
		status = http.StatusUnprocessableEntity
	}
	http.Error(w, err.Error(), status)
}
