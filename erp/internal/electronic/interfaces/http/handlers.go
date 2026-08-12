package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
	createDraft  application.CreateDraftUseCase
	confirm      application.ConfirmUseCase
	get          application.GetDocumentUseCase
	list         application.ListDocumentsUseCase
	numbering    application.ManageNumberingUseCase
	pdf          *application.GetDocumentPDFUseCase
	fromSale     *application.CreateFromSaleUseCase
	fromPurchase *application.CreateFromPurchaseUseCase
	dianRanges   *application.GetDianRangesUseCase
	sendEmail    *application.SendDocumentEmailUseCase
	audit        AuditLogger
}

func NewHandler(
	createDraft *application.CreateDraftUseCase,
	confirm *application.ConfirmUseCase,
	get *application.GetDocumentUseCase,
	list *application.ListDocumentsUseCase,
	numbering *application.ManageNumberingUseCase,
	pdf *application.GetDocumentPDFUseCase,
	fromSale *application.CreateFromSaleUseCase,
	fromPurchase *application.CreateFromPurchaseUseCase,
	dianRanges *application.GetDianRangesUseCase,
	sendEmail *application.SendDocumentEmailUseCase,
	audit AuditLogger,
) *Handler {
	return &Handler{
		createDraft:  *createDraft,
		confirm:      *confirm,
		get:          *get,
		list:         *list,
		numbering:    *numbering,
		pdf:          pdf,
		fromSale:     fromSale,
		fromPurchase: fromPurchase,
		dianRanges:   dianRanges,
		sendEmail:    sendEmail,
		audit:        audit,
	}
}

// ── DTOs de respuesta snake_case ─────────────────────────────────────────────────────────

type taxOutputDTO struct {
	TaxableAmountCents int64   `json:"taxable_amount_cents"`
	TaxAmountCents     int64   `json:"tax_amount_cents"`
	Percent            float64 `json:"percent"`
	TypeCode           string  `json:"type_code"`
	TypeName           string  `json:"type_name"`
}

type lineOutputDTO struct {
	Description        string         `json:"description"`
	Quantity           float64        `json:"quantity"`
	UnitCode           string         `json:"unit_code"`
	LineExtensionCents int64          `json:"line_extension_cents"`
	UnitPriceCents     int64          `json:"unit_price_cents"`
	FreeOfCharge       bool           `json:"free_of_charge,omitempty"`
	ItemCode           string         `json:"item_code,omitempty"`
	ItemTypeCode       string         `json:"item_type_code,omitempty"`
	ItemTypeName       string         `json:"item_type_name,omitempty"`
	ItemTypeAgencyID   string         `json:"item_type_agency_id,omitempty"`
	Taxes              []taxOutputDTO `json:"taxes,omitempty"`
}

type totalsOutputDTO struct {
	LineExtensionCents int64 `json:"line_extension_cents"`
	TaxExclusiveCents  int64 `json:"tax_exclusive_cents"`
	TaxInclusiveCents  int64 `json:"tax_inclusive_cents"`
	PrepaidCents       int64 `json:"prepaid_cents,omitempty"`
	PayableCents       int64 `json:"payable_cents"`
}

type billingRefOutputDTO struct {
	Prefix    string `json:"prefix"`
	Number    string `json:"number"`
	CUFE      string `json:"cufe"`
	IssueDate string `json:"issue_date"`
}

type discrepancyOutputDTO struct {
	ReferenceID  string `json:"reference_id"`
	ResponseCode string `json:"response_code"`
	Description  string `json:"description"`
}

type relatedNoteOutputDTO struct {
	ID                   uuid.UUID  `json:"id"`
	DianDocumentTypeCode string     `json:"dian_document_type_code"`
	Prefix               string     `json:"prefix,omitempty"`
	Number               int64      `json:"number,omitempty"`
	PayableCents         int64      `json:"payable_cents"`
	Status               string     `json:"status"`
	IssueDate            *time.Time `json:"issue_date,omitempty"`
}

type documentResponseDTO struct {
	ID                    uuid.UUID             `json:"id"`
	CompanyID             uuid.UUID             `json:"company_id"`
	NumberingRangeID      uuid.UUID             `json:"numbering_range_id"`
	DianDocumentTypeCode  string                `json:"dian_document_type_code"`
	Status                string                `json:"status"`
	Customer              partyInputDTO         `json:"customer"`
	Lines                 []lineOutputDTO       `json:"lines"`
	PaymentMeans          []paymentMeanInputDTO `json:"payment_means,omitempty"`
	Totals                totalsOutputDTO       `json:"totals"`
	Note                  string                `json:"note,omitempty"`
	CurrencyCode          string                `json:"currency_code,omitempty"`
	BillingReference      *billingRefOutputDTO  `json:"billing_reference,omitempty"`
	DiscrepancyResponse   *discrepancyOutputDTO `json:"discrepancy_response,omitempty"`
	NoteTypeCode          string                `json:"note_type_code,omitempty"`
	Supplier              *partyInputDTO        `json:"supplier,omitempty"`
	OperationTypeCode     string                `json:"operation_type_code,omitempty"`
	WithholdingTaxes      []taxOutputDTO        `json:"withholding_taxes,omitempty"`
	Prefix                string                `json:"prefix,omitempty"`
	Number                int64                 `json:"number,omitempty"`
	DocumentKey           string                `json:"document_key,omitempty"`
	IssueDate             string                `json:"issue_date,omitempty"`
	QRURL                 string                `json:"qr_url,omitempty"`
	DianTrackID           string                `json:"dian_track_id,omitempty"`
	DianStatusCode        string                `json:"dian_status_code,omitempty"`
	DianStatusDescription string                `json:"dian_status_description,omitempty"`
	DianStatusMessage     string                `json:"dian_status_message,omitempty"`
	CustomerID            *uuid.UUID            `json:"customer_id,omitempty"`
	SupplierID            *uuid.UUID            `json:"supplier_id,omitempty"`
	NCCount               int                   `json:"nc_count,omitempty"`
	NDCount               int                   `json:"nd_count,omitempty"`
	NACount               int                   `json:"na_count,omitempty"`
	// Visor didáctico — solo presentes en GET /{id} individual, nunca en el listado.
	RelatedNotes     []relatedNoteOutputDTO `json:"related_notes,omitempty"`
	NetPayableCents  *int64                 `json:"net_payable_cents,omitempty"`
	SourceDocumentID *uuid.UUID             `json:"source_document_id,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// computeNetPayable calcula el saldo neto de una FE/DS aplicando sus notas relacionadas.
// FE: payable − Σ NC aceptadas + Σ ND aceptadas. DS: payable − Σ NA aceptadas.
func computeNetPayable(payableCents int64, notes []domain.RelatedNote) *int64 {
	net := payableCents
	for _, n := range notes {
		if n.Status != domain.StatusAccepted {
			continue
		}
		switch n.DianDocumentTypeCode {
		case "91":
			net -= n.PayableCents
		case "92":
			net += n.PayableCents
		case "95":
			net -= n.PayableCents
		}
	}
	return &net
}

func partyToOutputDTO(p cofdom.Party) partyInputDTO {
	dto := partyInputDTO{
		EntityTypeCode:             p.EntityTypeCode,
		Identification:             identificationInputDTO{Number: p.Identification.Number, TypeCode: p.Identification.TypeCode, VerificationCode: p.Identification.VerificationCode},
		Name:                       p.Name,
		TaxSchemeCode:              p.TaxSchemeCode,
		TaxSchemeName:              p.TaxSchemeName,
		LiabilityCodes:             p.LiabilityCodes,
		TaxRegimeCode:              p.TaxRegimeCode,
		Phone:                      p.Phone,
		Email:                      p.Email,
		MerchantRegistrationNumber: p.MerchantRegistrationNumber,
	}
	if p.Address != (cofdom.Address{}) {
		dto.Address = &addressInputDTO{
			Line: p.Address.Line, CityCode: p.Address.CityCode, CityName: p.Address.CityName,
			PostalZone: p.Address.PostalZone, StateCode: p.Address.StateCode, StateName: p.Address.StateName,
			CountryCode: p.Address.CountryCode, CountryName: p.Address.CountryName,
		}
	}
	return dto
}

func taxToOutputDTO(t cofdom.Tax) taxOutputDTO {
	return taxOutputDTO{TaxableAmountCents: t.TaxableAmountCents, TaxAmountCents: t.TaxAmountCents, Percent: t.Percent, TypeCode: t.TypeCode, TypeName: t.TypeName}
}

func lineToOutputDTO(l cofdom.Line) lineOutputDTO {
	dto := lineOutputDTO{
		Description: l.Description, Quantity: l.Quantity, UnitCode: l.UnitCode,
		LineExtensionCents: l.LineExtensionCents, UnitPriceCents: l.UnitPriceCents,
		FreeOfCharge: l.FreeOfCharge, ItemCode: l.ItemCode,
		ItemTypeCode: l.ItemTypeCode, ItemTypeName: l.ItemTypeName, ItemTypeAgencyID: l.ItemTypeAgencyID,
	}
	dto.Taxes = make([]taxOutputDTO, len(l.Taxes))
	for i, t := range l.Taxes {
		dto.Taxes[i] = taxToOutputDTO(t)
	}
	return dto
}

func toDocumentResponseDTO(d *domain.Document) documentResponseDTO {
	dto := documentResponseDTO{
		ID: d.ID, CompanyID: d.CompanyID, NumberingRangeID: d.NumberingRangeID,
		DianDocumentTypeCode: d.DianDocumentTypeCode, Status: string(d.Status),
		Customer: partyToOutputDTO(d.Customer),
		Totals: totalsOutputDTO{
			LineExtensionCents: d.Totals.LineExtensionCents, TaxExclusiveCents: d.Totals.TaxExclusiveCents,
			TaxInclusiveCents: d.Totals.TaxInclusiveCents, PrepaidCents: d.Totals.PrepaidCents, PayableCents: d.Totals.PayableCents,
		},
		Note: d.Note, CurrencyCode: d.CurrencyCode, NoteTypeCode: d.NoteTypeCode,
		OperationTypeCode: d.OperationTypeCode, Prefix: d.Prefix, Number: d.Number,
		DocumentKey: d.DocumentKey, QRURL: d.QRURL,
		DianTrackID: d.DianTrackID, DianStatusCode: d.DianStatusCode,
		DianStatusDescription: d.DianStatusDescription, DianStatusMessage: d.DianStatusMessage,
		CustomerID: d.CustomerID, SupplierID: d.SupplierID,
		NCCount: d.NCCount, NDCount: d.NDCount, NACount: d.NACount,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
	if !d.IssueDate.IsZero() {
		dto.IssueDate = d.IssueDate.Format("2006-01-02")
	}
	dto.Lines = make([]lineOutputDTO, len(d.Lines))
	for i, l := range d.Lines {
		dto.Lines[i] = lineToOutputDTO(l)
	}
	dto.PaymentMeans = make([]paymentMeanInputDTO, len(d.PaymentMeans))
	for i, pm := range d.PaymentMeans {
		dto.PaymentMeans[i] = paymentMeanInputDTO{Code: pm.Code, PaymentMethodCode: pm.PaymentMethodCode, DueDate: pm.DueDate, PaymentReference: pm.PaymentReference}
	}
	if d.BillingReference != nil {
		dto.BillingReference = &billingRefOutputDTO{Prefix: d.BillingReference.Prefix, Number: d.BillingReference.Number, CUFE: d.BillingReference.CUFE, IssueDate: d.BillingReference.IssueDate}
	}
	if d.DiscrepancyResponse != nil {
		dto.DiscrepancyResponse = &discrepancyOutputDTO{ReferenceID: d.DiscrepancyResponse.ReferenceID, ResponseCode: d.DiscrepancyResponse.ResponseCode, Description: d.DiscrepancyResponse.Description}
	}
	if d.Supplier != nil {
		v := partyToOutputDTO(*d.Supplier)
		dto.Supplier = &v
	}
	if len(d.WithholdingTaxes) > 0 {
		dto.WithholdingTaxes = make([]taxOutputDTO, len(d.WithholdingTaxes))
		for i, t := range d.WithholdingTaxes {
			dto.WithholdingTaxes[i] = taxToOutputDTO(t)
		}
	}
	return dto
}

// ── Documentos ────────────────────────────────────────────────────────────────────────────

func (h *Handler) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	filter := domain.ListFilter{
		DianDocumentTypeCode: q.Get("dian_document_type_code"),
		Status:               domain.Status(q.Get("status")),
		Search:               q.Get("q"),
	}
	if s := q.Get("source_document_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			filter.SourceDocumentID = &id
		}
	}
	if s := q.Get("from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.From = t
		}
	}
	if s := q.Get("to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.To = t
		}
	}
	if s := q.Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			filter.Limit = n
		}
	}
	if s := q.Get("offset"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			filter.Offset = n
		}
	}
	docs, err := h.list.List(r.Context(), companyID, filter)
	if err != nil {
		writeErr(w, err)
		return
	}
	dtos := make([]documentResponseDTO, len(docs))
	for i, d := range docs {
		dtos[i] = toDocumentResponseDTO(d)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"documents": dtos,
		"count":     len(dtos),
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
	resp := toDocumentResponseDTO(doc)

	// Visor didáctico: FE (01) y DS (05) confirmados → notas relacionadas + saldo neto.
	if doc.Number != 0 && (doc.DianDocumentTypeCode == "01" || doc.DianDocumentTypeCode == "05") {
		if notes, err := h.get.GetRelatedNotes(r.Context(), companyID, doc.Prefix, doc.Number); err == nil && len(notes) > 0 {
			dtos := make([]relatedNoteOutputDTO, len(notes))
			for i, n := range notes {
				dtos[i] = relatedNoteOutputDTO{
					ID: n.ID, DianDocumentTypeCode: n.DianDocumentTypeCode,
					Prefix: n.Prefix, Number: n.Number,
					PayableCents: n.PayableCents, Status: string(n.Status),
					IssueDate: n.IssueDate,
				}
			}
			resp.RelatedNotes = dtos
			resp.NetPayableCents = computeNetPayable(doc.Totals.PayableCents, notes)
		}
	}

	// NC (91), ND (92), NA (95) → resolver ID del documento de origen por CUFE.
	if doc.BillingReference != nil && doc.BillingReference.CUFE != "" &&
		(doc.DianDocumentTypeCode == "91" || doc.DianDocumentTypeCode == "92" || doc.DianDocumentTypeCode == "95") {
		if src, err := h.get.GetByDocumentKey(r.Context(), companyID, doc.BillingReference.CUFE); err == nil {
			resp.SourceDocumentID = &src.ID
		}
	}

	writeJSON(w, http.StatusOK, resp)
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
	writeJSON(w, http.StatusOK, toDocumentResponseDTO(doc))
}

// handleCheckPendingStatus reintenta consultar la DIAN por un documento en StatusSent (envío
// asíncrono cuyo sondeo se agotó sin respuesta la primera vez) usando el zipKey ya guardado.
func (h *Handler) handleCheckPendingStatus(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	doc, err := h.confirm.CheckPendingStatus(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.status_checked", doc)
	writeJSON(w, http.StatusOK, toDocumentResponseDTO(doc))
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

func (h *Handler) handleSendDocumentEmail(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if h.sendEmail == nil {
		http.Error(w, "envío de correo no configurado en el servidor", http.StatusNotImplemented)
		return
	}
	if err := h.sendEmail.Send(r.Context(), companyID, id); err != nil {
		writeErr(w, err)
		return
	}
	if h.audit != nil {
		uid := tenant.GetUserID(r.Context())
		var userID *uuid.UUID
		if uid != uuid.Nil {
			userID = &uid
		}
		h.audit.Log(r.Context(), companyID, userID, "document.email_sent", "document", &id, nil)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) handleGetDocumentXML(w http.ResponseWriter, r *http.Request) {
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
	if doc.SignedXML == "" {
		http.Error(w, "el documento aún no tiene XML firmado", http.StatusNotFound)
		return
	}
	prefix := doc.Prefix + fmt.Sprintf("%d", doc.Number)
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xml"`, prefix))
	_, _ = w.Write([]byte(doc.SignedXML))
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
	writeJSON(w, http.StatusCreated, toDocumentResponseDTO(doc))
}

func (h *Handler) handleUpdateInvoiceDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
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
	doc, err := h.createDraft.UpdateInvoiceDraft(r.Context(), companyID, id, application.InvoiceDraftRequest{
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
	h.logDoc(r.Context(), companyID, "document.updated", doc)
	writeJSON(w, http.StatusOK, toDocumentResponseDTO(doc))
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
	h.logDoc(r.Context(), companyID, "document.created", doc)
	writeJSON(w, http.StatusCreated, toDocumentResponseDTO(doc))
}

func (h *Handler) handleUpdateCreditNoteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
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
	doc, err := h.createDraft.UpdateCreditNoteDraft(r.Context(), companyID, id, application.NoteDraftRequest{
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
	h.logDoc(r.Context(), companyID, "document.updated", doc)
	writeJSON(w, http.StatusOK, toDocumentResponseDTO(doc))
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
	h.logDoc(r.Context(), companyID, "document.created", doc)
	writeJSON(w, http.StatusCreated, toDocumentResponseDTO(doc))
}

func (h *Handler) handleUpdateDebitNoteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
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
	doc, err := h.createDraft.UpdateDebitNoteDraft(r.Context(), companyID, id, application.NoteDraftRequest{
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
	h.logDoc(r.Context(), companyID, "document.updated", doc)
	writeJSON(w, http.StatusOK, toDocumentResponseDTO(doc))
}

type supportDocBody struct {
	NumberingRangeID  uuid.UUID             `json:"numbering_range_id"`
	Supplier          partyInputDTO         `json:"supplier"`
	Lines             []lineInputDTO        `json:"lines"`
	PaymentMeans      []paymentMeanInputDTO `json:"payment_means"`
	Note              string                `json:"note"`
	CurrencyCode      string                `json:"currency_code"`
	OperationTypeCode string                `json:"operation_type_code"`
	WithholdingTaxes  []taxInputDTO         `json:"withholding_taxes"`
	SupplierID        *uuid.UUID            `json:"supplier_id"`
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
		Supplier:          body.Supplier.toCofdom(),
		Lines:             lines,
		PaymentMeans:      paymentMeansInputToCofdom(body.PaymentMeans),
		Note:              body.Note,
		CurrencyCode:      body.CurrencyCode,
		OperationTypeCode: body.OperationTypeCode,
		WithholdingTaxes:  taxesInputToCofdom(body.WithholdingTaxes),
		SupplierID:        body.SupplierID,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.created", doc)
	writeJSON(w, http.StatusCreated, toDocumentResponseDTO(doc))
}

func (h *Handler) handleUpdateSupportDocDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
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
	doc, err := h.createDraft.UpdateSupportDocumentDraft(r.Context(), companyID, id, application.SupportDocumentDraftRequest{
		CompanyID:         companyID,
		NumberingRangeID:  body.NumberingRangeID,
		Supplier:          body.Supplier.toCofdom(),
		Lines:             lines,
		PaymentMeans:      paymentMeansInputToCofdom(body.PaymentMeans),
		Note:              body.Note,
		CurrencyCode:      body.CurrencyCode,
		OperationTypeCode: body.OperationTypeCode,
		WithholdingTaxes:  taxesInputToCofdom(body.WithholdingTaxes),
		SupplierID:        body.SupplierID,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.updated", doc)
	writeJSON(w, http.StatusOK, toDocumentResponseDTO(doc))
}

type adjustmentNoteBody struct {
	NumberingRangeID    uuid.UUID                        `json:"numbering_range_id"`
	Supplier            partyInputDTO                    `json:"supplier"`
	Lines               []lineInputDTO                   `json:"lines"`
	PaymentMeans        []paymentMeanInputDTO            `json:"payment_means"`
	Note                string                           `json:"note"`
	CurrencyCode        string                           `json:"currency_code"`
	OperationTypeCode   string                           `json:"operation_type_code"`
	WithholdingTaxes    []taxInputDTO                    `json:"withholding_taxes"`
	SupplierID          *uuid.UUID                       `json:"supplier_id"`
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
		Supplier:            body.Supplier.toCofdom(),
		Lines:               lines,
		PaymentMeans:        paymentMeansInputToCofdom(body.PaymentMeans),
		Note:                body.Note,
		CurrencyCode:        body.CurrencyCode,
		OperationTypeCode:   body.OperationTypeCode,
		WithholdingTaxes:    taxesInputToCofdom(body.WithholdingTaxes),
		SupplierID:          body.SupplierID,
		BillingReference:    body.BillingReference,
		DiscrepancyResponse: body.DiscrepancyResponse,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.created", doc)
	writeJSON(w, http.StatusCreated, toDocumentResponseDTO(doc))
}

func (h *Handler) handleUpdateAdjustmentNoteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
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
	doc, err := h.createDraft.UpdateAdjustmentNoteDraft(r.Context(), companyID, id, application.AdjustmentNoteDraftRequest{
		CompanyID:           companyID,
		NumberingRangeID:    body.NumberingRangeID,
		Supplier:            body.Supplier.toCofdom(),
		Lines:               lines,
		PaymentMeans:        paymentMeansInputToCofdom(body.PaymentMeans),
		Note:                body.Note,
		CurrencyCode:        body.CurrencyCode,
		OperationTypeCode:   body.OperationTypeCode,
		WithholdingTaxes:    taxesInputToCofdom(body.WithholdingTaxes),
		SupplierID:          body.SupplierID,
		BillingReference:    body.BillingReference,
		DiscrepancyResponse: body.DiscrepancyResponse,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.updated", doc)
	writeJSON(w, http.StatusOK, toDocumentResponseDTO(doc))
}

func (h *Handler) handleCloneDocument(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	doc, err := h.createDraft.CloneDraft(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.cloned", doc)
	writeJSON(w, http.StatusCreated, toDocumentResponseDTO(doc))
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
	h.logDoc(r.Context(), companyID, "document.created", doc)
	writeJSON(w, http.StatusCreated, toDocumentResponseDTO(doc))
}

type fromPurchaseBody struct {
	NumberingRangeID  uuid.UUID `json:"numbering_range_id"`
	OperationTypeCode string    `json:"operation_type_code"`
}

func (h *Handler) handleCreateSupportDocFromPurchase(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	purchaseID, err := uuid.Parse(r.PathValue("purchase_id"))
	if err != nil {
		http.Error(w, "purchase_id inválido", http.StatusBadRequest)
		return
	}
	var body fromPurchaseBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	doc, err := h.fromPurchase.Execute(r.Context(), application.FromPurchaseRequest{
		CompanyID:         companyID,
		PurchaseID:        purchaseID,
		NumberingRangeID:  body.NumberingRangeID,
		OperationTypeCode: body.OperationTypeCode,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.created", doc)
	writeJSON(w, http.StatusCreated, toDocumentResponseDTO(doc))
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

// formatDate SIEMPRE formatea en UTC -- resolution_date/valid_from/valid_to de un rango de
// numeración se parsean con time.Parse("2006-01-02", ...) (medianoche UTC), y pgx devuelve el
// TIMESTAMPTZ ya escaneado en la zona LOCAL del proceso Go. Sin el .UTC() explícito acá, en
// cualquier servidor con zona horaria detrás de UTC (Colombia, UTC-5) la medianoche UTC se ve
// como las 19:00 del día anterior en hora local, y Format("2006-01-02") imprime la fecha
// equivocada -- mismo bug encontrado 2026-08-11 en sales/purchase, aplica igual acá porque
// comparte el patrón exacto de parseo.
func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
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
	companyID, ok := requireManage(w, r)
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
	if h.audit != nil {
		uid := tenant.GetUserID(r.Context())
		var userID *uuid.UUID
		if uid != uuid.Nil {
			userID = &uid
		}
		h.audit.Log(r.Context(), companyID, userID, "numbering_range.created", "numbering_range", &created.ID, map[string]any{
			"dian_document_type_code": created.DianDocumentTypeCode,
			"prefix":                  created.Prefix,
			"resolution_number":       created.ResolutionNumber,
		})
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
	companyID, ok := requireManage(w, r)
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
	if h.audit != nil {
		uid := tenant.GetUserID(r.Context())
		var userID *uuid.UUID
		if uid != uuid.Nil {
			userID = &uid
		}
		h.audit.Log(r.Context(), companyID, userID, "numbering_range.deactivated", "numbering_range", &id, nil)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleActivateRange(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireManage(w, r)
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
	if h.audit != nil {
		uid := tenant.GetUserID(r.Context())
		var userID *uuid.UUID
		if uid != uuid.Nil {
			userID = &uid
		}
		h.audit.Log(r.Context(), companyID, userID, "numbering_range.activated", "numbering_range", &id, nil)
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
	} else if doc.Supplier != nil && doc.Supplier.Name != "" {
		meta["supplier_name"] = doc.Supplier.Name
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

// requireManage exige además rol owner/admin -- para mutaciones sobre rangos de numeración DIAN
// (recurso legalmente sensible: una resolución mal asignada puede invalidar documentos emitidos).
func requireManage(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := mustTenant(w, r)
	if !ok {
		return uuid.Nil, false
	}
	if !tenant.CanManage(r.Context()) {
		http.Error(w, "requiere rol de administrador", http.StatusForbidden)
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
		errors.Is(err, domain.ErrDocumentNotPending),
		errors.Is(err, domain.ErrEmptyLines),
		errors.Is(err, domain.ErrMissingCustomer),
		errors.Is(err, domain.ErrMissingPaymentMeans),
		errors.Is(err, domain.ErrMissingBillingReference),
		errors.Is(err, domain.ErrMissingSupplier),
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
