package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/application"
	"github.com/diegofxm/erp/internal/accounting/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

type Handler struct {
	post    *application.PostJournalUseCase
	get     *application.GetJournalUseCase
	void    *application.VoidJournalUseCase
	period  *application.ManagePeriodUseCase
	accounts domain.AccountRepository
}

func NewHandler(
	post *application.PostJournalUseCase,
	get *application.GetJournalUseCase,
	void *application.VoidJournalUseCase,
	period *application.ManagePeriodUseCase,
	accounts domain.AccountRepository,
) *Handler {
	return &Handler{post: post, get: get, void: void, period: period, accounts: accounts}
}

// ── Asientos ─────────────────────────────────────────────────────────────────

func (h *Handler) handleListJournals(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	list, err := h.get.List(r.Context(), cid, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []*domain.JournalEntry{}
	}
	respond(w, http.StatusOK, list)
}

func (h *Handler) handleGetJournal(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	entry, err := h.get.ByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrJournalNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, entry)
}

func (h *Handler) handlePostJournal(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}

	var body struct {
		Date               string                  `json:"date"`
		Description        string                  `json:"description"`
		Source             string                  `json:"source"`
		EntryType          string                  `json:"entry_type"`
		VoucherType        string                  `json:"voucher_type"`
		SourceDocumentID   string                  `json:"source_document_id"`
		SourceDocumentType string                  `json:"source_document_type"`
		Book               string                  `json:"book"`
		Lines              []postLineBody          `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	date, err := time.Parse("2006-01-02", body.Date)
	if err != nil {
		respondError(w, http.StatusBadRequest, "date debe ser YYYY-MM-DD")
		return
	}

	req := application.PostJournalRequest{
		CompanyID:          cid,
		Date:               date,
		Description:        body.Description,
		Source:             body.Source,
		EntryType:          body.EntryType,
		VoucherType:        body.VoucherType,
		SourceDocumentType: body.SourceDocumentType,
		Book:               body.Book,
	}
	if body.SourceDocumentID != "" {
		req.SourceDocumentID, _ = uuid.Parse(body.SourceDocumentID)
	}
	req.Lines = make([]application.PostLineRequest, len(body.Lines))
	for i, l := range body.Lines {
		req.Lines[i] = application.PostLineRequest{
			AccountCode:     l.AccountCode,
			DebitCents:      l.DebitCents,
			CreditCents:     l.CreditCents,
			ThirdPartyNIT:   l.ThirdPartyNIT,
			CostCenter:      l.CostCenter,
			Description:     l.Description,
			ForeignAmount:   l.ForeignAmount,
			ForeignCurrency: l.ForeignCurrency,
		}
	}

	entry, err := h.post.Execute(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrEmptyLines) || errors.Is(err, domain.ErrInvalidLine) || errors.Is(err, domain.ErrImbalancedEntry) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if errors.Is(err, domain.ErrPeriodClosed) || errors.Is(err, domain.ErrAccountNotFound) || errors.Is(err, domain.ErrAccountNotPosting) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, entry)
}

type postLineBody struct {
	AccountCode     string `json:"account_code"`
	DebitCents      int64  `json:"debit_cents"`
	CreditCents     int64  `json:"credit_cents"`
	ThirdPartyNIT   string `json:"third_party_nit"`
	CostCenter      string `json:"cost_center"`
	Description     string `json:"description"`
	ForeignAmount   int64  `json:"foreign_amount"`
	ForeignCurrency string `json:"foreign_currency"`
}

func (h *Handler) handleVoidJournal(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.void.Execute(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrJournalNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrJournalVoided) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "void"})
}

// ── Cuentas ───────────────────────────────────────────────────────────────────

func (h *Handler) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := h.accounts.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Account{}
	}
	respond(w, http.StatusOK, list)
}

func (h *Handler) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	acct, err := h.accounts.GetByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, acct)
}

// ── Períodos ──────────────────────────────────────────────────────────────────

func (h *Handler) handleListPeriods(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.period.List(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.AccountingPeriod{}
	}
	respond(w, http.StatusOK, list)
}

func (h *Handler) handleClosePeriod(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.period.Close(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrPeriodNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrPeriodClosed) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "closed"})
}

// ── Reportes ──────────────────────────────────────────────────────────────────

func (h *Handler) handlePLReport(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil || year < 2000 {
		respondError(w, http.StatusBadRequest, "year requerido (YYYY)")
		return
	}
	balances, err := h.get.PLBalances(r.Context(), cid, year)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if balances == nil {
		balances = []domain.PLBalance{}
	}
	respond(w, http.StatusOK, balances)
}

func (h *Handler) handleBSReport(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	asOfStr := r.URL.Query().Get("as_of")
	asOf, err := time.Parse("2006-01-02", asOfStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "as_of requerido (YYYY-MM-DD)")
		return
	}
	balances, err := h.get.BSBalances(r.Context(), cid, asOf)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if balances == nil {
		balances = []domain.PLBalance{}
	}
	respond(w, http.StatusOK, balances)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func requireTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid := tenant.GetCompanyID(r.Context())
	if cid == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "empresa activa requerida")
		return uuid.Nil, false
	}
	return cid, true
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}
