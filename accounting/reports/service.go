package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service ejecuta queries de reportes directamente sobre el pool.
// Los reportes son siempre calculados — no se guardan balances.
type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// TrialBalance devuelve el balance de comprobación para una empresa en un rango de fechas.
func (s *Service) TrialBalance(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]*TrialBalanceRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			a.code,
			a.name,
			a.category,
			COALESCE(SUM(jl.debit), 0)  AS total_debit,
			COALESCE(SUM(jl.credit), 0) AS total_credit,
			COALESCE(SUM(jl.debit) - SUM(jl.credit), 0) AS balance
		FROM accounting.accounts a
		JOIN accounting.journal_lines jl ON jl.account_id = a.id
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		WHERE je.company_id = $1
		  AND je.status = 'POSTED'
		  AND je.date >= $2
		  AND je.date <= $3
		GROUP BY a.id, a.code, a.name, a.category
		ORDER BY a.code`,
		companyID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("trial balance: %w", err)
	}
	defer rows.Close()

	var out []*TrialBalanceRow
	for rows.Next() {
		var r TrialBalanceRow
		if err := rows.Scan(&r.AccountCode, &r.AccountName, &r.Category,
			&r.TotalDebit, &r.TotalCredit, &r.Balance); err != nil {
			return nil, fmt.Errorf("scan trial balance: %w", err)
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// GeneralLedger devuelve el libro mayor de una cuenta con saldo acumulado.
func (s *Service) GeneralLedger(ctx context.Context, companyID uuid.UUID, accountCode string, from, to time.Time) ([]*LedgerRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			je.id::text,
			je.date::text,
			je.description,
			jl.debit,
			jl.credit
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		JOIN accounting.accounts a ON a.id = jl.account_id
		WHERE je.company_id = $1
		  AND je.status = 'POSTED'
		  AND a.code = $2
		  AND je.date >= $3
		  AND je.date <= $4
		ORDER BY je.date, je.created_at`,
		companyID, accountCode, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("general ledger: %w", err)
	}
	defer rows.Close()

	var out []*LedgerRow
	var running float64
	for rows.Next() {
		var r LedgerRow
		if err := rows.Scan(&r.JournalID, &r.Date, &r.Description, &r.Debit, &r.Credit); err != nil {
			return nil, fmt.Errorf("scan ledger: %w", err)
		}
		running += r.Debit - r.Credit
		r.RunningBal = running
		out = append(out, &r)
	}
	return out, rows.Err()
}

// IncomeStatement devuelve el estado de resultados para un rango de fechas.
func (s *Service) IncomeStatement(ctx context.Context, companyID uuid.UUID, from, to time.Time) (*IncomeStatement, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			a.code,
			a.name,
			a.category,
			COALESCE(SUM(jl.credit) - SUM(jl.debit), 0) AS amount
		FROM accounting.accounts a
		JOIN accounting.journal_lines jl ON jl.account_id = a.id
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		WHERE je.company_id = $1
		  AND je.status = 'POSTED'
		  AND je.date >= $2
		  AND je.date <= $3
		  AND a.category IN ('Ingresos', 'Gastos', 'Costos')
		GROUP BY a.id, a.code, a.name, a.category
		ORDER BY a.category, a.code`,
		companyID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("income statement: %w", err)
	}
	defer rows.Close()

	is := &IncomeStatement{}
	for rows.Next() {
		var r IncomeStatementRow
		if err := rows.Scan(&r.AccountCode, &r.AccountName, &r.Category, &r.Amount); err != nil {
			return nil, fmt.Errorf("scan income statement: %w", err)
		}
		switch r.Category {
		case "Ingresos":
			is.Revenue = append(is.Revenue, r)
			is.NetIncome += r.Amount
		default: // Gastos, Costos
			is.Expenses = append(is.Expenses, r)
			is.NetIncome -= r.Amount
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return is, nil
}

// BalanceSheet devuelve el balance general a una fecha de corte.
func (s *Service) BalanceSheet(ctx context.Context, companyID uuid.UUID, asOf time.Time) (*BalanceSheet, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			a.code,
			a.name,
			a.category,
			COALESCE(SUM(jl.debit) - SUM(jl.credit), 0) AS balance
		FROM accounting.accounts a
		JOIN accounting.journal_lines jl ON jl.account_id = a.id
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		WHERE je.company_id = $1
		  AND je.status = 'POSTED'
		  AND je.date <= $2
		  AND a.category IN ('Activo', 'Pasivo', 'Patrimonio')
		GROUP BY a.id, a.code, a.name, a.category
		ORDER BY a.category, a.code`,
		companyID, asOf.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("balance sheet: %w", err)
	}
	defer rows.Close()

	bs := &BalanceSheet{}
	for rows.Next() {
		var r BalanceSheetRow
		var category string
		if err := rows.Scan(&r.AccountCode, &r.AccountName, &category, &r.Balance); err != nil {
			return nil, fmt.Errorf("scan balance sheet: %w", err)
		}
		switch category {
		case "Activo":
			bs.Assets = append(bs.Assets, r)
			bs.TotalAssets += r.Balance
		case "Pasivo":
			bs.Liabilities = append(bs.Liabilities, r)
			bs.TotalLiabilitiesAndEquity += r.Balance
		case "Patrimonio":
			bs.Equity = append(bs.Equity, r)
			bs.TotalLiabilitiesAndEquity += r.Balance
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bs, nil
}
