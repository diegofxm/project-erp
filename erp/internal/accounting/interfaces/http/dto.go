package http

import (
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/application"
	"github.com/diegofxm/erp/internal/accounting/domain"
)

// Package http domain structs no llevan json tags (dominio puro) — este archivo
// define los DTOs snake_case que sí se serializan, siguiendo el mismo patrón
// usado en sales/purchase/inventory.

type accountDTO struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	ParentCode string `json:"parent_code"`
	Level      int    `json:"level"`
	Category   string `json:"category"`
	IsPosting  bool   `json:"is_posting"`
	IsActive   bool   `json:"is_active"`
}

func toAccountDTO(a domain.Account) accountDTO {
	return accountDTO{
		ID: a.ID.String(), Code: a.Code, Name: a.Name, ParentCode: a.ParentCode,
		Level: a.Level, Category: a.Category, IsPosting: a.IsPosting, IsActive: a.IsActive,
	}
}

func toAccountDTOs(list []domain.Account) []accountDTO {
	out := make([]accountDTO, len(list))
	for i, a := range list {
		out[i] = toAccountDTO(a)
	}
	return out
}

type periodDTO struct {
	ID        string     `json:"id"`
	CompanyID string     `json:"company_id"`
	Year      int        `json:"year"`
	Month     int        `json:"month"`
	Status    string     `json:"status"`
	OpenedAt  time.Time  `json:"opened_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
}

func toPeriodDTO(p domain.AccountingPeriod) periodDTO {
	return periodDTO{
		ID: p.ID.String(), CompanyID: p.CompanyID.String(), Year: p.Year, Month: p.Month,
		Status: string(p.Status), OpenedAt: p.OpenedAt, ClosedAt: p.ClosedAt,
	}
}

func toPeriodDTOs(list []domain.AccountingPeriod) []periodDTO {
	out := make([]periodDTO, len(list))
	for i, p := range list {
		out[i] = toPeriodDTO(p)
	}
	return out
}

type withholdingConceptDTO struct {
	ID                string  `json:"id"`
	Code              string  `json:"code"`
	Name              string  `json:"name"`
	Type              string  `json:"type"`
	RateBP            int     `json:"rate_bp"`
	MinBaseUVT        float64 `json:"min_base_uvt"`
	AccountPayable    string  `json:"account_payable"`
	AccountReceivable string  `json:"account_receivable"`
	ApplicableTo      string  `json:"applicable_to"`
}

func toWithholdingConceptDTOs(list []domain.WithholdingConcept) []withholdingConceptDTO {
	out := make([]withholdingConceptDTO, len(list))
	for i, c := range list {
		out[i] = withholdingConceptDTO{
			ID: c.ID.String(), Code: c.Code, Name: c.Name, Type: c.Type, RateBP: c.RateBP,
			MinBaseUVT: c.MinBaseUVT, AccountPayable: c.AccountPayable,
			AccountReceivable: c.AccountReceivable, ApplicableTo: c.ApplicableTo,
		}
	}
	return out
}

type withholdingCertificateDTO struct {
	ID            string  `json:"id"`
	FiscalYear    int     `json:"fiscal_year"`
	ThirdPartyNIT string  `json:"third_party_nit"`
	ConceptCode   string  `json:"concept_code"`
	ConceptName   string  `json:"concept_name"`
	WHType        string  `json:"wh_type"`
	GrossAmount   float64 `json:"gross_amount"`
	TaxWithheld   float64 `json:"tax_withheld"`
	Status        string  `json:"status"`
	IssuedAt      string  `json:"issued_at,omitempty"`
}

func toCertificateDTOs(list []domain.WithholdingCertificate) []withholdingCertificateDTO {
	out := make([]withholdingCertificateDTO, len(list))
	for i, c := range list {
		dto := withholdingCertificateDTO{
			ID: c.ID.String(), FiscalYear: c.FiscalYear, ThirdPartyNIT: c.ThirdPartyNIT,
			ConceptCode: c.ConceptCode, ConceptName: c.ConceptName, WHType: c.WHType,
			GrossAmount: c.GrossAmount, TaxWithheld: c.TaxWithheld, Status: c.Status,
		}
		if c.IssuedAt != nil {
			dto.IssuedAt = c.IssuedAt.Format("2006-01-02")
		}
		out[i] = dto
	}
	return out
}

type bankAccountDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BankName  string `json:"bank_name"`
	AccountNo string `json:"account_no"`
	AccountID string `json:"account_id"`
	IsActive  bool   `json:"is_active"`
}

func toBankAccountDTO(a domain.BankAccount) bankAccountDTO {
	return bankAccountDTO{ID: a.ID.String(), Name: a.Name, BankName: a.BankName, AccountNo: a.AccountNo, AccountID: a.AccountID.String(), IsActive: a.IsActive}
}

func toBankAccountDTOs(list []domain.BankAccount) []bankAccountDTO {
	out := make([]bankAccountDTO, len(list))
	for i, a := range list {
		out[i] = toBankAccountDTO(a)
	}
	return out
}

type statementLineDTO struct {
	ID            string  `json:"id"`
	BankAccountID string  `json:"bank_account_id"`
	Date          string  `json:"date"`
	Description   string  `json:"description"`
	Debit         int64   `json:"debit"`
	Credit        int64   `json:"credit"`
	Reference     string  `json:"reference"`
	IsReconciled  bool    `json:"is_reconciled"`
	JournalLineID *string `json:"journal_line_id,omitempty"`
}

func toStatementLineDTO(l domain.BankStatementLine) statementLineDTO {
	dto := statementLineDTO{
		ID: l.ID.String(), BankAccountID: l.BankAccountID.String(), Date: l.Date.Format("2006-01-02"),
		Description: l.Description, Debit: l.Debit, Credit: l.Credit, Reference: l.Reference, IsReconciled: l.IsReconciled,
	}
	if l.JournalLineID != nil {
		s := l.JournalLineID.String()
		dto.JournalLineID = &s
	}
	return dto
}

func toStatementLineDTOs(list []domain.BankStatementLine) []statementLineDTO {
	out := make([]statementLineDTO, len(list))
	for i, l := range list {
		out[i] = toStatementLineDTO(l)
	}
	return out
}

type candidateDTO struct {
	LineID        string `json:"line_id"`
	JournalID     string `json:"journal_id"`
	Date          string `json:"date"`
	Description   string `json:"description"`
	VoucherType   string `json:"voucher_type"`
	VoucherNumber string `json:"voucher_number"`
	Debit         int64  `json:"debit"`
	Credit        int64  `json:"credit"`
}

func toCandidateDTOs(list []domain.ReconciliationCandidate) []candidateDTO {
	out := make([]candidateDTO, len(list))
	for i, c := range list {
		out[i] = candidateDTO{
			LineID: c.LineID.String(), JournalID: c.JournalID.String(), Date: c.Date.Format("2006-01-02"),
			Description: c.Description, VoucherType: c.VoucherType, VoucherNumber: c.VoucherNumber,
			Debit: c.Debit, Credit: c.Credit,
		}
	}
	return out
}

type fixedAssetDTO struct {
	ID                  string `json:"id"`
	Code                string `json:"code"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	AssetAccount        string `json:"asset_account"`
	DepreciationAccount string `json:"depreciation_account"`
	AccumulatedAccount  string `json:"accumulated_account"`
	AcquisitionDate     string `json:"acquisition_date"`
	AcquisitionCost     int64  `json:"acquisition_cost"`
	SalvageValue        int64  `json:"salvage_value"`
	UsefulLifeMonths    int    `json:"useful_life_months"`
	MonthlyDepreciation int64  `json:"monthly_depreciation"`
	Accumulated         int64  `json:"accumulated"`
	Status              string `json:"status"`
	ThirdPartyNIT       string `json:"third_party_nit,omitempty"`
}

func toFixedAssetDTO(a application.AssetWithAccumulated) fixedAssetDTO {
	asset := a.FixedAsset
	return fixedAssetDTO{
		ID: asset.ID.String(), Code: asset.Code, Name: asset.Name, Description: asset.Description,
		AssetAccount: asset.AssetAccount, DepreciationAccount: asset.DepreciationAccount, AccumulatedAccount: asset.AccumulatedAccount,
		AcquisitionDate: asset.AcquisitionDate.Format("2006-01-02"), AcquisitionCost: asset.AcquisitionCost, SalvageValue: asset.SalvageValue,
		UsefulLifeMonths: asset.UsefulLifeMonths, MonthlyDepreciation: asset.MonthlyDepreciation(), Accumulated: a.Accumulated,
		Status: string(asset.Status), ThirdPartyNIT: asset.ThirdPartyNIT,
	}
}

func toFixedAssetDTOs(list []application.AssetWithAccumulated) []fixedAssetDTO {
	out := make([]fixedAssetDTO, len(list))
	for i, a := range list {
		out[i] = toFixedAssetDTO(a)
	}
	return out
}

type depreciationRunDTO struct {
	ID        string  `json:"id"`
	RunDate   string  `json:"run_date"`
	Status    string  `json:"status"`
	JournalID *string `json:"journal_id,omitempty"`
}

func toDepreciationRunDTOs(list []domain.DepreciationRun) []depreciationRunDTO {
	out := make([]depreciationRunDTO, len(list))
	for i, r := range list {
		dto := depreciationRunDTO{ID: r.ID.String(), RunDate: r.RunDate.Format("2006-01-02"), Status: string(r.Status)}
		if r.JournalID != nil {
			s := r.JournalID.String()
			dto.JournalID = &s
		}
		out[i] = dto
	}
	return out
}

type budgetDTO struct {
	ID     string `json:"id"`
	Year   int    `json:"year"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func toBudgetDTO(b domain.Budget) budgetDTO {
	return budgetDTO{ID: b.ID.String(), Year: b.Year, Name: b.Name, Status: string(b.Status)}
}

func toBudgetDTOs(list []domain.Budget) []budgetDTO {
	out := make([]budgetDTO, len(list))
	for i, b := range list {
		out[i] = toBudgetDTO(b)
	}
	return out
}

type budgetLineDTO struct {
	AccountCode string    `json:"account_code"`
	AccountName string    `json:"account_name"`
	Months      [12]int64 `json:"months"`
	Total       int64     `json:"total"`
}

func toBudgetLineDTOs(list []domain.BudgetLine) []budgetLineDTO {
	out := make([]budgetLineDTO, len(list))
	for i, l := range list {
		out[i] = budgetLineDTO{AccountCode: l.AccountCode, AccountName: l.AccountName, Months: l.Months, Total: l.Total()}
	}
	return out
}

type budgetActualDTO struct {
	AccountCode    string    `json:"account_code"`
	AccountName    string    `json:"account_name"`
	BudgetedMonths [12]int64 `json:"budgeted_months"`
	ActualMonths   [12]int64 `json:"actual_months"`
}

func toBudgetActualDTOs(list []domain.BudgetActualRow) []budgetActualDTO {
	out := make([]budgetActualDTO, len(list))
	for i, r := range list {
		out[i] = budgetActualDTO{AccountCode: r.AccountCode, AccountName: r.AccountName, BudgetedMonths: r.BudgetedMonths, ActualMonths: r.ActualMonths}
	}
	return out
}

type ivaDeclarationDTO struct {
	ID              string `json:"id"`
	PeriodStart     string `json:"period_start"`
	PeriodEnd       string `json:"period_end"`
	PeriodType      string `json:"period_type"`
	GeneratedIVA    int64  `json:"generated_iva"`
	DeductibleIVA   int64  `json:"deductible_iva"`
	NetIVA          int64  `json:"net_iva"`
	PreviousBalance int64  `json:"previous_balance"`
	AmountToPay     int64  `json:"amount_to_pay"`
	CarryForward    int64  `json:"carry_forward"`
	Status          string `json:"status"`
}

func toIVADTOs(list []domain.IVADeclaration) []ivaDeclarationDTO {
	out := make([]ivaDeclarationDTO, len(list))
	for i, d := range list {
		out[i] = ivaDeclarationDTO{
			ID: d.ID.String(), PeriodStart: d.PeriodStart.Format("2006-01-02"), PeriodEnd: d.PeriodEnd.Format("2006-01-02"),
			PeriodType: string(d.PeriodType), GeneratedIVA: d.GeneratedIVA, DeductibleIVA: d.DeductibleIVA, NetIVA: d.NetIVA,
			PreviousBalance: d.PreviousBalance, AmountToPay: d.AmountToPay, CarryForward: d.CarryForward, Status: string(d.Status),
		}
	}
	return out
}

type incomeTaxDeclarationDTO struct {
	ID            string `json:"id"`
	FiscalYear    int    `json:"fiscal_year"`
	TaxableIncome int64  `json:"taxable_income"`
	TaxRateBP     int    `json:"tax_rate_bp"`
	TaxComputed   int64  `json:"tax_computed"`
	TaxToPay      int64  `json:"tax_to_pay"`
	AmountDue     int64  `json:"amount_due"`
	Status        string `json:"status"`
}

func toIncomeTaxDTOs(list []domain.IncomeTaxDeclaration) []incomeTaxDeclarationDTO {
	out := make([]incomeTaxDeclarationDTO, len(list))
	for i, d := range list {
		out[i] = incomeTaxDeclarationDTO{
			ID: d.ID.String(), FiscalYear: d.FiscalYear, TaxableIncome: d.TaxableIncome, TaxRateBP: d.TaxRateBP,
			TaxComputed: d.TaxComputed, TaxToPay: d.TaxToPay, AmountDue: d.AmountDue, Status: string(d.Status),
		}
	}
	return out
}

type icaTariffDTO struct {
	ID               string `json:"id"`
	MunicipalityCode string `json:"municipality_code"`
	CIIUCode         string `json:"ciiu_code"`
	FiscalYear       int    `json:"fiscal_year"`
	RateBP           int    `json:"rate_bp"`
	SurchargeBP      int    `json:"surcharge_bp"`
}

func toICATariffDTOs(list []domain.ICATariff) []icaTariffDTO {
	out := make([]icaTariffDTO, len(list))
	for i, t := range list {
		out[i] = icaTariffDTO{ID: t.ID.String(), MunicipalityCode: t.MunicipalityCode, CIIUCode: t.CIIUCode, FiscalYear: t.FiscalYear, RateBP: t.RateBP, SurchargeBP: t.SurchargeBP}
	}
	return out
}

type icaDeclarationDTO struct {
	ID               string `json:"id"`
	MunicipalityCode string `json:"municipality_code"`
	PeriodStart      string `json:"period_start"`
	PeriodEnd        string `json:"period_end"`
	CIIUCode         string `json:"ciiu_code"`
	GrossRevenue     int64  `json:"gross_revenue"`
	NetBase          int64  `json:"net_base"`
	TaxComputed      int64  `json:"tax_computed"`
	SurchargeAmount  int64  `json:"surcharge_amount"`
	TaxToPay         int64  `json:"tax_to_pay"`
	PreviousBalance  int64  `json:"previous_balance"`
	AmountDue        int64  `json:"amount_due"`
	CarryForward     int64  `json:"carry_forward"`
	Status           string `json:"status"`
}

func toICADTOs(list []domain.ICADeclaration) []icaDeclarationDTO {
	out := make([]icaDeclarationDTO, len(list))
	for i, d := range list {
		out[i] = icaDeclarationDTO{
			ID: d.ID.String(), MunicipalityCode: d.MunicipalityCode, PeriodStart: d.PeriodStart.Format("2006-01-02"),
			PeriodEnd: d.PeriodEnd.Format("2006-01-02"), CIIUCode: d.CIIUCode, GrossRevenue: d.GrossRevenue, NetBase: d.NetBase,
			TaxComputed: d.TaxComputed, SurchargeAmount: d.SurchargeAmount, TaxToPay: d.TaxToPay,
			PreviousBalance: d.PreviousBalance, AmountDue: d.AmountDue, CarryForward: d.CarryForward, Status: string(d.Status),
		}
	}
	return out
}

type voucherTypeDTO struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	ResetsAnnually bool   `json:"resets_annually"`
	IsActive       bool   `json:"is_active"`
}

func toVoucherTypeDTOs(list []*domain.VoucherTypeConfig) []voucherTypeDTO {
	out := make([]voucherTypeDTO, len(list))
	for i, c := range list {
		out[i] = voucherTypeDTO{ID: c.ID.String(), Code: c.Code, Name: c.Name, ResetsAnnually: c.ResetsAnnually, IsActive: c.IsActive}
	}
	return out
}

type journalLineDTO struct {
	ID              string `json:"id"`
	AccountID       string `json:"account_id"`
	AccountCode     string `json:"account_code"`
	Debit           int64  `json:"debit"`
	Credit          int64  `json:"credit"`
	ThirdPartyNIT   string `json:"third_party_nit"`
	CostCenter      string `json:"cost_center"`
	Description     string `json:"description"`
	ForeignAmount   int64  `json:"foreign_amount"`
	ForeignCurrency string `json:"foreign_currency"`
}

type journalEntryDTO struct {
	ID                 string           `json:"id"`
	CompanyID          string           `json:"company_id"`
	PeriodID           string           `json:"period_id"`
	Date               time.Time        `json:"date"`
	Description        string           `json:"description"`
	Status             string           `json:"status"`
	Source             string           `json:"source"`
	EntryType          string           `json:"entry_type"`
	VoucherType        string           `json:"voucher_type"`
	VoucherNumber      string           `json:"voucher_number"`
	SourceDocumentID   string           `json:"source_document_id,omitempty"`
	SourceDocumentType string           `json:"source_document_type,omitempty"`
	Book               string           `json:"book"`
	Lines              []journalLineDTO `json:"lines"`
	CreatedAt          time.Time        `json:"created_at"`
}

func toJournalEntryDTO(e *domain.JournalEntry) journalEntryDTO {
	lines := make([]journalLineDTO, len(e.Lines))
	for i, l := range e.Lines {
		lines[i] = journalLineDTO{
			ID: l.ID.String(), AccountID: l.AccountID.String(), AccountCode: l.AccountCode,
			Debit: l.Debit, Credit: l.Credit, ThirdPartyNIT: l.ThirdPartyNIT,
			CostCenter: l.CostCenter, Description: l.Description,
			ForeignAmount: l.ForeignAmount, ForeignCurrency: l.ForeignCurrency,
		}
	}
	dto := journalEntryDTO{
		ID: e.ID.String(), CompanyID: e.CompanyID.String(), PeriodID: e.PeriodID.String(),
		Date: e.Date, Description: e.Description, Status: string(e.Status), Source: e.Source,
		EntryType: string(e.EntryType), VoucherType: e.VoucherType, VoucherNumber: e.VoucherNumber,
		SourceDocumentType: e.SourceDocumentType, Book: string(e.Book), Lines: lines, CreatedAt: e.CreatedAt,
	}
	if e.SourceDocumentID != uuid.Nil {
		dto.SourceDocumentID = e.SourceDocumentID.String()
	}
	return dto
}

func toJournalEntryDTOs(list []*domain.JournalEntry) []journalEntryDTO {
	out := make([]journalEntryDTO, len(list))
	for i, e := range list {
		out[i] = toJournalEntryDTO(e)
	}
	return out
}

type balanceDTO struct {
	AccountID   string `json:"account_id"`
	AccountCode string `json:"account_code"`
	AccountName string `json:"account_name"`
	Category    string `json:"category"`
	Balance     int64  `json:"balance"`
}

func toBalanceDTOs(list []domain.PLBalance) []balanceDTO {
	out := make([]balanceDTO, len(list))
	for i, b := range list {
		out[i] = balanceDTO{
			AccountID: b.AccountID.String(), AccountCode: b.AccountCode,
			AccountName: b.AccountName, Category: b.Category, Balance: b.Balance,
		}
	}
	return out
}

type trialBalanceRowDTO struct {
	AccountID   string `json:"account_id"`
	AccountCode string `json:"account_code"`
	AccountName string `json:"account_name"`
	Category    string `json:"category"`
	Debit       int64  `json:"debit"`
	Credit      int64  `json:"credit"`
	Balance     int64  `json:"balance"`
}

func toTrialBalanceDTOs(list []domain.TrialBalanceRow) []trialBalanceRowDTO {
	out := make([]trialBalanceRowDTO, len(list))
	for i, r := range list {
		out[i] = trialBalanceRowDTO{
			AccountID: r.AccountID.String(), AccountCode: r.AccountCode, AccountName: r.AccountName,
			Category: r.Category, Debit: r.Debit, Credit: r.Credit, Balance: r.Balance,
		}
	}
	return out
}

type ledgerLineDTO struct {
	JournalID      string    `json:"journal_id"`
	Date           time.Time `json:"date"`
	Description    string    `json:"description"`
	VoucherType    string    `json:"voucher_type"`
	VoucherNumber  string    `json:"voucher_number"`
	Debit          int64     `json:"debit"`
	Credit         int64     `json:"credit"`
	RunningBalance int64     `json:"running_balance"`
}

func toLedgerLineDTOs(list []domain.LedgerLine) []ledgerLineDTO {
	out := make([]ledgerLineDTO, len(list))
	for i, l := range list {
		out[i] = ledgerLineDTO{
			JournalID: l.JournalID.String(), Date: l.Date, Description: l.Description,
			VoucherType: l.VoucherType, VoucherNumber: l.VoucherNumber,
			Debit: l.Debit, Credit: l.Credit, RunningBalance: l.RunningBalance,
		}
	}
	return out
}
