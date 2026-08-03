package http

import (
	"time"

	"github.com/google/uuid"

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
