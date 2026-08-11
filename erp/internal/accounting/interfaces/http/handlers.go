package http

import (
	"context"
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

// AuditLogger es la interfaz mínima para registrar eventos de auditoría.
// La implementa audit/application.UseCase sin requerir importar ese paquete.
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

type Handler struct {
	post         *application.PostJournalUseCase
	get          *application.GetJournalUseCase
	void         *application.VoidJournalUseCase
	period       *application.ManagePeriodUseCase
	accounts     domain.AccountRepository
	withholding  domain.WithholdingConceptRepository
	certificates *application.IssueWithholdingCertificatesUseCase
	bank         *application.ManageBankUseCase
	assets       *application.FixedAssetUseCase
	dispose      *application.DisposeFixedAssetUseCase
	depreciation *application.RunDepreciationUseCase
	budget       *application.BudgetUseCase
	iva          *application.IVAUseCase
	incomeTax    *application.IncomeTaxUseCase
	ica          *application.ICAUseCase
	exchangeRate *application.ExchangeRateUseCase
	reconcile    *application.ReconciliationUseCase
	audit        AuditLogger
}

func NewHandler(
	post *application.PostJournalUseCase,
	get *application.GetJournalUseCase,
	void *application.VoidJournalUseCase,
	period *application.ManagePeriodUseCase,
	accounts domain.AccountRepository,
	withholding domain.WithholdingConceptRepository,
	certificates *application.IssueWithholdingCertificatesUseCase,
	bank *application.ManageBankUseCase,
	assets *application.FixedAssetUseCase,
	dispose *application.DisposeFixedAssetUseCase,
	depreciation *application.RunDepreciationUseCase,
	budget *application.BudgetUseCase,
	iva *application.IVAUseCase,
	incomeTax *application.IncomeTaxUseCase,
	ica *application.ICAUseCase,
	exchangeRate *application.ExchangeRateUseCase,
	reconcile *application.ReconciliationUseCase,
	audit AuditLogger,
) *Handler {
	return &Handler{
		post: post, get: get, void: void, period: period, accounts: accounts, withholding: withholding,
		certificates: certificates, bank: bank, assets: assets, dispose: dispose, depreciation: depreciation, budget: budget,
		iva: iva, incomeTax: incomeTax, ica: ica, exchangeRate: exchangeRate, reconcile: reconcile, audit: audit,
	}
}

func (h *Handler) logAudit(ctx context.Context, companyID uuid.UUID, action, resourceType string, id uuid.UUID, meta map[string]any) {
	if h.audit == nil {
		return
	}
	uid := tenant.GetUserID(ctx)
	var userID *uuid.UUID
	if uid != uuid.Nil {
		userID = &uid
	}
	h.audit.Log(ctx, companyID, userID, action, resourceType, &id, meta)
}

// ── Declaración de IVA ───────────────────────────────────────────────────────

func (h *Handler) handleGenerateIVA(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		PeriodStart string `json:"period_start"`
		PeriodEnd   string `json:"period_end"`
		PeriodType  string `json:"period_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	from, err1 := time.Parse("2006-01-02", body.PeriodStart)
	to, err2 := time.Parse("2006-01-02", body.PeriodEnd)
	if err1 != nil || err2 != nil {
		respondError(w, http.StatusBadRequest, "period_start/period_end deben ser YYYY-MM-DD")
		return
	}
	d, err := h.iva.Generate(r.Context(), cid, from, to, body.PeriodType)
	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toIVADTOs([]domain.IVADeclaration{*d})[0])
}

func (h *Handler) handleListIVA(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.iva.List(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toIVADTOs(list))
}

func (h *Handler) handleFileIVA(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.iva.MarkFiled(r.Context(), cid, id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "declaration.iva_filed", "iva_declaration", id, nil)
	respond(w, http.StatusOK, map[string]string{"status": "filed"})
}

// ── Declaración de Renta ─────────────────────────────────────────────────────

func (h *Handler) handleGenerateIncomeTax(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		FiscalYear int `json:"fiscal_year"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	d, err := h.incomeTax.Generate(r.Context(), cid, body.FiscalYear)
	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toIncomeTaxDTOs([]domain.IncomeTaxDeclaration{*d})[0])
}

func (h *Handler) handleListIncomeTax(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.incomeTax.List(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toIncomeTaxDTOs(list))
}

func (h *Handler) handleFileIncomeTax(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.incomeTax.MarkFiled(r.Context(), cid, id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "declaration.income_tax_filed", "income_tax_declaration", id, nil)
	respond(w, http.StatusOK, map[string]string{"status": "filed"})
}

// ── Declaración de ICA ───────────────────────────────────────────────────────

func (h *Handler) handleSetICATariff(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireTenant(w, r); !ok {
		return
	}
	var body struct {
		MunicipalityCode string `json:"municipality_code"`
		CIIUCode         string `json:"ciiu_code"`
		FiscalYear       int    `json:"fiscal_year"`
		RateBP           int    `json:"rate_bp"`
		SurchargeBP      int    `json:"surcharge_bp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	t, err := h.ica.SetTariff(r.Context(), body.MunicipalityCode, body.CIIUCode, body.FiscalYear, body.RateBP, body.SurchargeBP)
	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toICATariffDTOs([]domain.ICATariff{*t})[0])
}

func (h *Handler) handleListICATariffs(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireTenant(w, r); !ok {
		return
	}
	list, err := h.ica.ListTariffs(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toICATariffDTOs(list))
}

func (h *Handler) handleGenerateICA(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		MunicipalityCode string `json:"municipality_code"`
		CIIUCode         string `json:"ciiu_code"`
		PeriodStart      string `json:"period_start"`
		PeriodEnd        string `json:"period_end"`
		PeriodType       string `json:"period_type"`
		DeductionsCents  int64  `json:"deductions_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	from, err1 := time.Parse("2006-01-02", body.PeriodStart)
	to, err2 := time.Parse("2006-01-02", body.PeriodEnd)
	if err1 != nil || err2 != nil {
		respondError(w, http.StatusBadRequest, "period_start/period_end deben ser YYYY-MM-DD")
		return
	}
	d, err := h.ica.Generate(r.Context(), cid, application.GenerateICARequest{
		MunicipalityCode: body.MunicipalityCode, CIIUCode: body.CIIUCode, PeriodStart: from, PeriodEnd: to,
		PeriodType: body.PeriodType, DeductionsCents: body.DeductionsCents,
	})
	if err != nil {
		if errors.Is(err, domain.ErrICATariffNotFound) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toICADTOs([]domain.ICADeclaration{*d})[0])
}

func (h *Handler) handleListICA(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.ica.List(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toICADTOs(list))
}

func (h *Handler) handleFileICA(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.ica.MarkFiled(r.Context(), cid, id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "declaration.ica_filed", "ica_declaration", id, nil)
	respond(w, http.StatusOK, map[string]string{"status": "filed"})
}

// ── Presupuestos ─────────────────────────────────────────────────────────────

func (h *Handler) handleCreateBudget(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		Year int    `json:"year"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	b, err := h.budget.Create(r.Context(), cid, body.Year, body.Name)
	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toBudgetDTO(*b))
}

func (h *Handler) handleListBudgets(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	list, err := h.budget.List(r.Context(), cid, year)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toBudgetDTOs(list))
}

func (h *Handler) handleSetBudgetLine(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	budgetID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		AccountCode string    `json:"account_code"`
		Months      [12]int64 `json:"months"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	line, err := h.budget.SetLine(r.Context(), cid, application.SetBudgetLineRequest{BudgetID: budgetID, AccountCode: body.AccountCode, Months: body.Months})
	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toBudgetLineDTOs([]domain.BudgetLine{*line})[0])
}

func (h *Handler) handleListBudgetLines(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	budgetID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.budget.ListLines(r.Context(), cid, budgetID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toBudgetLineDTOs(list))
}

func (h *Handler) handleBudgetCompare(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	budgetID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.budget.Compare(r.Context(), cid, budgetID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toBudgetActualDTOs(list))
}

func (h *Handler) handleApproveBudget(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	budgetID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.budget.SetStatus(r.Context(), cid, budgetID, domain.BudgetApproved); err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "approved"})
}

// ── Activos fijos ────────────────────────────────────────────────────────────

func (h *Handler) handleCreateFixedAsset(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		Code                 string `json:"code"`
		Name                 string `json:"name"`
		Description          string `json:"description"`
		AssetAccount         string `json:"asset_account"`
		DepreciationAccount  string `json:"depreciation_account"`
		AccumulatedAccount   string `json:"accumulated_account"`
		GainAccount          string `json:"gain_account"`
		LossAccount          string `json:"loss_account"`
		AcquisitionDate      string `json:"acquisition_date"`
		AcquisitionCostCents int64  `json:"acquisition_cost_cents"`
		SalvageValueCents    int64  `json:"salvage_value_cents"`
		UsefulLifeMonths     int    `json:"useful_life_months"`
		ThirdPartyNIT        string `json:"third_party_nit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	date, err := time.Parse("2006-01-02", body.AcquisitionDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "acquisition_date debe ser YYYY-MM-DD")
		return
	}
	a, err := h.assets.Create(r.Context(), cid, application.CreateFixedAssetRequest{
		Code: body.Code, Name: body.Name, Description: body.Description,
		AssetAccount: body.AssetAccount, DepreciationAccount: body.DepreciationAccount, AccumulatedAccount: body.AccumulatedAccount,
		GainAccount: body.GainAccount, LossAccount: body.LossAccount,
		AcquisitionDate: date, AcquisitionCostCents: body.AcquisitionCostCents, SalvageValueCents: body.SalvageValueCents,
		UsefulLifeMonths: body.UsefulLifeMonths, ThirdPartyNIT: body.ThirdPartyNIT,
	})
	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "fixed_asset.created", "fixed_asset", a.ID, map[string]any{"code": a.Code, "name": a.Name})
	respond(w, http.StatusCreated, toFixedAssetDTO(application.AssetWithAccumulated{FixedAsset: *a}))
}

func (h *Handler) handleListFixedAssets(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.assets.ListWithAccumulated(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toFixedAssetDTOs(list))
}

func (h *Handler) handleDisposeFixedAsset(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		DisposalDate        string `json:"disposal_date"`
		ProceedsCents       int64  `json:"proceeds_cents"`
		ProceedsAccountCode string `json:"proceeds_account_code"`
		Description         string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	date, err := time.Parse("2006-01-02", body.DisposalDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "disposal_date debe ser YYYY-MM-DD")
		return
	}
	a, err := h.dispose.Execute(r.Context(), cid, id, application.DisposeFixedAssetRequest{
		DisposalDate: date, ProceedsCents: body.ProceedsCents,
		ProceedsAccountCode: body.ProceedsAccountCode, Description: body.Description,
	})
	if err != nil {
		if errors.Is(err, domain.ErrFixedAssetNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrAssetAlreadyDisposed) || errors.Is(err, domain.ErrPeriodClosed) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "fixed_asset.disposed", "fixed_asset", a.ID, map[string]any{
		"code": a.Code, "name": a.Name, "proceeds_cents": body.ProceedsCents,
	})
	respond(w, http.StatusOK, toFixedAssetDTO(application.AssetWithAccumulated{FixedAsset: *a}))
}

func (h *Handler) handleRunDepreciation(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		Date string `json:"date"`
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
	run, err := h.depreciation.Execute(r.Context(), cid, date)
	if err != nil {
		if errors.Is(err, domain.ErrDepreciationExists) || errors.Is(err, domain.ErrNoDepreciableAssets) || errors.Is(err, domain.ErrPeriodClosed) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "depreciation.run", "depreciation_run", run.ID, nil)
	respond(w, http.StatusCreated, toDepreciationRunDTOs([]domain.DepreciationRun{*run})[0])
}

func (h *Handler) handleListDepreciationRuns(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.depreciation.ListRuns(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toDepreciationRunDTOs(list))
}

// ── Bancos ────────────────────────────────────────────────────────────────────

func (h *Handler) handleCreateBankAccount(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		Name        string `json:"name"`
		BankName    string `json:"bank_name"`
		AccountNo   string `json:"account_no"`
		AccountCode string `json:"account_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	a, err := h.bank.CreateAccount(r.Context(), cid, application.CreateBankAccountRequest{
		Name: body.Name, BankName: body.BankName, AccountNo: body.AccountNo, AccountCode: body.AccountCode,
	})
	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toBankAccountDTO(*a))
}

func (h *Handler) handleListBankAccounts(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.bank.ListAccounts(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toBankAccountDTOs(list))
}

func (h *Handler) handleUpdateBankAccount(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Name      string `json:"name"`
		BankName  string `json:"bank_name"`
		AccountNo string `json:"account_no"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	a, err := h.bank.UpdateAccount(r.Context(), cid, id, application.UpdateBankAccountRequest{
		Name: body.Name, BankName: body.BankName, AccountNo: body.AccountNo,
	})
	if err != nil {
		if errors.Is(err, domain.ErrBankAccountNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusOK, toBankAccountDTO(*a))
}

func (h *Handler) handleDeactivateBankAccount(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.bank.DeactivateAccount(r.Context(), cid, id); err != nil {
		if errors.Is(err, domain.ErrBankAccountNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAddStatementLine(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	bankAccountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Date        string `json:"date"`
		Description string `json:"description"`
		DebitCents  int64  `json:"debit_cents"`
		CreditCents int64  `json:"credit_cents"`
		Reference   string `json:"reference"`
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
	l, err := h.bank.AddStatementLine(r.Context(), cid, bankAccountID, application.AddStatementLineRequest{
		Date: date, Description: body.Description, DebitCents: body.DebitCents, CreditCents: body.CreditCents, Reference: body.Reference,
	})
	if err != nil {
		if errors.Is(err, domain.ErrBankAccountNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toStatementLineDTO(*l))
}

func (h *Handler) handleListStatement(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	bankAccountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.bank.ListStatement(r.Context(), cid, bankAccountID)
	if err != nil {
		if errors.Is(err, domain.ErrBankAccountNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toStatementLineDTOs(list))
}

func (h *Handler) handleCandidates(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	lineID, err := uuid.Parse(r.PathValue("line_id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "line_id inválido")
		return
	}
	list, err := h.bank.Candidates(r.Context(), cid, lineID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toCandidateDTOs(list))
}

func (h *Handler) handleReconcile(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	lineID, err := uuid.Parse(r.PathValue("line_id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "line_id inválido")
		return
	}
	var body struct {
		JournalLineID uuid.UUID `json:"journal_line_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if err := h.bank.Reconcile(r.Context(), cid, lineID, body.JournalLineID); err != nil {
		if errors.Is(err, domain.ErrAlreadyReconciled) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "reconciled"})
}

func (h *Handler) handleListWithholdingConcepts(w http.ResponseWriter, r *http.Request) {
	list, err := h.withholding.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toWithholdingConceptDTOs(list))
}

func (h *Handler) handleIssueCertificates(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		SupplierID    uuid.UUID `json:"supplier_id"`
		ThirdPartyNIT string    `json:"third_party_nit"`
		FiscalYear    int       `json:"fiscal_year"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	list, err := h.certificates.Execute(r.Context(), cid, application.IssueCertificatesRequest{
		SupplierID: body.SupplierID, ThirdPartyNIT: body.ThirdPartyNIT, FiscalYear: body.FiscalYear,
	})
	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toCertificateDTOs(list))
}

func (h *Handler) handleListCertificates(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil || year < 2000 {
		respondError(w, http.StatusBadRequest, "year requerido (YYYY)")
		return
	}
	list, err := h.certificates.List(r.Context(), cid, year)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toCertificateDTOs(list))
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
	respond(w, http.StatusOK, toJournalEntryDTOs(list))
}

func (h *Handler) handleGetJournal(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	entry, err := h.get.ByID(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrJournalNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toJournalEntryDTO(entry))
}

func (h *Handler) handlePostJournal(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}

	var body struct {
		Date               string         `json:"date"`
		Description        string         `json:"description"`
		Source             string         `json:"source"`
		EntryType          string         `json:"entry_type"`
		VoucherType        string         `json:"voucher_type"`
		SourceDocumentID   string         `json:"source_document_id"`
		SourceDocumentType string         `json:"source_document_type"`
		Book               string         `json:"book"`
		Lines              []postLineBody `json:"lines"`
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
		if errors.Is(err, domain.ErrPeriodClosed) || errors.Is(err, domain.ErrAccountNotFound) || errors.Is(err, domain.ErrAccountNotPosting) || errors.Is(err, domain.ErrVoucherTypeUnknown) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), req.CompanyID, "journal.posted", "journal", entry.ID, map[string]any{
		"voucher_type": entry.VoucherType, "voucher_number": entry.VoucherNumber,
	})
	respond(w, http.StatusCreated, toJournalEntryDTO(entry))
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
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.void.Execute(r.Context(), cid, id); err != nil {
		if errors.Is(err, domain.ErrJournalNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrJournalVoided) || errors.Is(err, domain.ErrPeriodClosed) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "journal.voided", "journal", id, nil)
	respond(w, http.StatusOK, map[string]string{"status": "void"})
}

// ── Cuentas ───────────────────────────────────────────────────────────────────

func (h *Handler) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := h.accounts.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toAccountDTOs(list))
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
	respond(w, http.StatusOK, toAccountDTO(*acct))
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
	respond(w, http.StatusOK, toPeriodDTOs(list))
}

func (h *Handler) handleClosePeriod(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.period.Close(r.Context(), cid, id); err != nil {
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
	h.logAudit(r.Context(), cid, "period.closed", "period", id, nil)
	respond(w, http.StatusOK, map[string]string{"status": "closed"})
}

func (h *Handler) handleReopenPeriod(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.period.Reopen(r.Context(), cid, id); err != nil {
		if errors.Is(err, domain.ErrPeriodNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "period.reopened", "period", id, nil)
	respond(w, http.StatusOK, map[string]string{"status": "open"})
}

// ── Tipos de comprobante ─────────────────────────────────────────────────────

func (h *Handler) handleCreateVoucherType(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		Code           string `json:"code"`
		Name           string `json:"name"`
		ResetsAnnually bool   `json:"resets_annually"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	cfg, err := h.get.RegisterVoucherType(r.Context(), domain.VoucherTypeConfig{
		CompanyID: cid, Code: body.Code, Name: body.Name, ResetsAnnually: body.ResetsAnnually, IsActive: true,
	})
	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toVoucherTypeDTOs([]*domain.VoucherTypeConfig{cfg})[0])
}

// handleSetVoucherCounter fija el próximo consecutivo a emitir para un tipo de comprobante —
// pensado para migrar una empresa que ya traía su propia numeración de otro sistema. Solo
// admin/owner.
func (h *Handler) handleSetVoucherCounter(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireManage(w, r)
	if !ok {
		return
	}
	var body struct {
		Code       string `json:"code"`
		Year       int    `json:"year"`
		NextNumber int    `json:"next_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	seq, err := h.get.SetVoucherCounter(r.Context(), cid, body.Code, body.Year, body.NextNumber)
	if err != nil {
		if errors.Is(err, domain.ErrNumberCounterInvalid) || errors.Is(err, domain.ErrNumberCounterBackwards) ||
			errors.Is(err, domain.ErrVoucherTypeUnknown) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"code": body.Code, "year": body.Year, "last_seq": seq, "next_will_be": seq + 1,
	})
}

func (h *Handler) handleListVoucherTypes(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.get.ListVoucherTypes(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toVoucherTypeDTOs(list))
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
	respond(w, http.StatusOK, toBalanceDTOs(balances))
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
	respond(w, http.StatusOK, toBalanceDTOs(balances))
}

func (h *Handler) handleTrialBalance(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	from, to, ok := parseDateRange(w, r)
	if !ok {
		return
	}
	rows, err := h.get.TrialBalance(r.Context(), cid, from, to)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toTrialBalanceDTOs(rows))
}

func (h *Handler) handleAccountLedger(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	code := r.PathValue("code")
	from, to, ok := parseDateRange(w, r)
	if !ok {
		return
	}
	if _, err := h.accounts.GetByCode(r.Context(), code); err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	lines, err := h.get.AccountLedger(r.Context(), cid, code, from, to)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toLedgerLineDTOs(lines))
}

func parseDateRange(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "from requerido (YYYY-MM-DD)")
		return time.Time{}, time.Time{}, false
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "to requerido (YYYY-MM-DD)")
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}

// ── Tasas de cambio (TRM) ────────────────────────────────────────────────────
// No están ligadas a la empresa (es un dato de mercado) — solo exige sesión autenticada.

type exchangeRateDTO struct {
	RateDate     string  `json:"rate_date"`
	FromCurrency string  `json:"from_currency"`
	ToCurrency   string  `json:"to_currency"`
	Rate         float64 `json:"rate"`
	Source       string  `json:"source"`
	Description  string  `json:"description"`
}

func toExchangeRateDTO(e domain.ExchangeRate) exchangeRateDTO {
	return exchangeRateDTO{
		RateDate: e.RateDate.Format("2006-01-02"), FromCurrency: e.FromCurrency,
		ToCurrency: e.ToCurrency, Rate: e.Rate(), Source: e.Source, Description: e.Description,
	}
}

func (h *Handler) handleSetExchangeRate(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireTenant(w, r); !ok {
		return
	}
	var body struct {
		RateDate     string  `json:"rate_date"`
		FromCurrency string  `json:"from_currency"`
		ToCurrency   string  `json:"to_currency"`
		Rate         float64 `json:"rate"`
		Source       string  `json:"source"`
		Description  string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	date, err := time.Parse("2006-01-02", body.RateDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "rate_date debe ser YYYY-MM-DD")
		return
	}
	rate, err := h.exchangeRate.Set(r.Context(), application.SetExchangeRateRequest{
		RateDate: date, FromCurrency: body.FromCurrency, ToCurrency: body.ToCurrency,
		Rate: body.Rate, Source: body.Source, Description: body.Description,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, toExchangeRateDTO(*rate))
}

func (h *Handler) handleSyncExchangeRate(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireTenant(w, r); !ok {
		return
	}
	rate, err := h.exchangeRate.Sync(r.Context())
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respond(w, http.StatusCreated, toExchangeRateDTO(*rate))
}

// handleLookupExchangeRate resuelve la TRM de una fecha puntual — primero contra la base local
// (?date=YYYY-MM-DD, ?from=USD opcional); si no existe todavía, la consulta una sola vez al
// servicio TRM propio y la guarda (ver ExchangeRateUseCase.GetOrFetch). Pensada para que el
// contador busque cualquier fecha pasada sin esperar al disparador diario.
func (h *Handler) handleLookupExchangeRate(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireTenant(w, r); !ok {
		return
	}
	dateStr := r.URL.Query().Get("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "date debe ser YYYY-MM-DD")
		return
	}
	from := r.URL.Query().Get("from")
	rate, err := h.exchangeRate.GetOrFetch(r.Context(), date, from, "")
	if err != nil {
		if errors.Is(err, domain.ErrExchangeRateNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respond(w, http.StatusOK, toExchangeRateDTO(*rate))
}

// handleListExchangeRates pagina el historial (más reciente primero) — nunca trae toda la tabla
// de una vez. ?limit=7&offset=0 por defecto.
func (h *Handler) handleListExchangeRates(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireTenant(w, r); !ok {
		return
	}
	limit := 7
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	list, total, err := h.exchangeRate.List(r.Context(), limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]exchangeRateDTO, len(list))
	for i, e := range list {
		dtos[i] = toExchangeRateDTO(e)
	}
	respond(w, http.StatusOK, map[string]any{"rates": dtos, "total": total})
}

// handleGetTodayExchangeRate es solo lectura contra la base de datos (nunca toca el servicio
// externo) — para mostrar la TRM de hoy junto al título del panel sin importar la paginación de
// la lista de abajo ni disparar ninguna sincronización solo por abrir la página.
func (h *Handler) handleGetTodayExchangeRate(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireTenant(w, r); !ok {
		return
	}
	rate, err := h.exchangeRate.GetToday(r.Context())
	if err != nil {
		if errors.Is(err, domain.ErrExchangeRateNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toExchangeRateDTO(*rate))
}

// ── Conciliación de cuentas (cruce de partidas) ─────────────────────────────

type openLineDTO struct {
	LineID        uuid.UUID `json:"line_id"`
	JournalID     uuid.UUID `json:"journal_id"`
	Date          string    `json:"date"`
	Description   string    `json:"description"`
	VoucherNumber string    `json:"voucher_number"`
	ThirdPartyNIT string    `json:"third_party_nit"`
	DebitCents    int64     `json:"debit_cents"`
	CreditCents   int64     `json:"credit_cents"`
}

func toOpenLineDTO(l domain.OpenLine) openLineDTO {
	return openLineDTO{
		LineID: l.LineID, JournalID: l.JournalID, Date: l.Date.Format("2006-01-02"),
		Description: l.Description, VoucherNumber: l.VoucherNumber, ThirdPartyNIT: l.ThirdPartyNIT,
		DebitCents: l.Debit, CreditCents: l.Credit,
	}
}

func (h *Handler) handleListOpenLines(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	code := r.URL.Query().Get("account_code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "account_code requerido")
		return
	}
	list, err := h.reconcile.OpenLines(r.Context(), cid, code)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]openLineDTO, len(list))
	for i, l := range list {
		dtos[i] = toOpenLineDTO(l)
	}
	respond(w, http.StatusOK, dtos)
}

func (h *Handler) handleMarkReconciled(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		JournalLineID  uuid.UUID  `json:"journal_line_id"`
		ReconciledWith *uuid.UUID `json:"reconciled_with"`
		Note           string     `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	mark, err := h.reconcile.Mark(r.Context(), cid, body.JournalLineID, body.ReconciledWith, body.Note)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "reconciliation.marked", "journal_line", body.JournalLineID, nil)
	respond(w, http.StatusCreated, mark)
}

func (h *Handler) handleUnmarkReconciled(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	lineID, err := uuid.Parse(r.PathValue("line_id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "line_id inválido")
		return
	}
	if err := h.reconcile.Unmark(r.Context(), cid, lineID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "reconciliation.unmarked", "journal_line", lineID, nil)
	w.WriteHeader(http.StatusNoContent)
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

func requireManage(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return uuid.Nil, false
	}
	if !tenant.CanManage(r.Context()) {
		respondError(w, http.StatusForbidden, "requiere rol de administrador")
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
