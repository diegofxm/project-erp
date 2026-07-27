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

// bookClause genera la cláusula SQL para filtrar por libro contable.
// "PCGA" incluye asientos PCGA y BOTH; "NIIF" incluye NIIF y BOTH; cualquier otro valor = sin filtro.
func bookClause(book string) string {
	switch book {
	case "PCGA":
		return " AND je.book IN ('PCGA', 'BOTH')"
	case "NIIF":
		return " AND je.book IN ('NIIF', 'BOTH')"
	default:
		return ""
	}
}

// resolveBook extrae el primer valor de books o devuelve "" si no se pasa ninguno.
func resolveBook(books []string) string {
	if len(books) > 0 {
		return books[0]
	}
	return ""
}

// TrialBalance devuelve el balance de comprobación para una empresa en un rango de fechas.
// El parámetro opcional book ("PCGA" | "NIIF") filtra por libro contable; vacío = ambos libros.
func (s *Service) TrialBalance(ctx context.Context, companyID uuid.UUID, from, to time.Time, book ...string) ([]*TrialBalanceRow, error) {
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
		  AND je.date <= $3`+bookClause(resolveBook(book))+`
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
	var running int64
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
// El parámetro opcional book ("PCGA" | "NIIF") filtra por libro contable; vacío = ambos libros.
func (s *Service) IncomeStatement(ctx context.Context, companyID uuid.UUID, from, to time.Time, book ...string) (*IncomeStatement, error) {
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
		  AND a.category IN ('Ingresos', 'Gastos', 'Costos', 'Costo de Producción')`+bookClause(resolveBook(book))+`
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
		// Amount = SUM(credit) - SUM(debit):
		//   Ingresos → positivo (créditos > débitos): suma a utilidad.
		//   Gastos/Costos → negativo (débitos > créditos): también suma (resta utilidad).
		switch r.Category {
		case "Ingresos":
			is.Revenue = append(is.Revenue, r)
		default: // Gastos, Costos, Costo de Producción
			is.Expenses = append(is.Expenses, r)
		}
		is.NetIncome += r.Amount
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return is, nil
}

// AuxiliaryByThird devuelve el libro auxiliar de una cuenta para un NIT específico,
// con saldo acumulado. Es el reporte estándar de libro auxiliar por tercero en Colombia.
func (s *Service) AuxiliaryByThird(ctx context.Context, companyID uuid.UUID, accountCode, terceroNIT string, from, to time.Time) ([]*AuxiliaryRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			je.id::text,
			je.date::text,
			je.description,
			COALESCE(jl.third_party_nit, ''),
			jl.debit,
			jl.credit
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		JOIN accounting.accounts a ON a.id = jl.account_id
		WHERE je.company_id = $1
		  AND je.status     = 'POSTED'
		  AND a.code        = $2
		  AND jl.third_party_nit = $3
		  AND je.date       >= $4
		  AND je.date       <= $5
		ORDER BY je.date, je.created_at`,
		companyID, accountCode, terceroNIT, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("auxiliary by third: %w", err)
	}
	defer rows.Close()

	var out []*AuxiliaryRow
	var running int64
	for rows.Next() {
		var r AuxiliaryRow
		if err := rows.Scan(&r.JournalID, &r.Date, &r.Description, &r.TerceroNIT, &r.Debit, &r.Credit); err != nil {
			return nil, fmt.Errorf("scan auxiliary: %w", err)
		}
		running += r.Debit - r.Credit
		r.RunningBal = running
		out = append(out, &r)
	}
	return out, rows.Err()
}

// TerceroBalance devuelve el saldo por tercero (NIT) en una cuenta para un rango de fechas.
// Útil para cartera (cuenta 13xx) y proveedores (cuenta 22xx).
func (s *Service) TerceroBalance(ctx context.Context, companyID uuid.UUID, accountCode string, from, to time.Time) ([]*TerceroBalanceRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			jl.third_party_nit,
			COALESCE(SUM(jl.debit), 0)                        AS total_debit,
			COALESCE(SUM(jl.credit), 0)                       AS total_credit,
			COALESCE(SUM(jl.debit) - SUM(jl.credit), 0)      AS balance
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		JOIN accounting.accounts a ON a.id = jl.account_id
		WHERE je.company_id  = $1
		  AND je.status      = 'POSTED'
		  AND a.code         = $2
		  AND je.date       >= $3
		  AND je.date       <= $4
		  AND jl.third_party_nit IS NOT NULL
		GROUP BY jl.third_party_nit
		HAVING COALESCE(SUM(jl.debit) - SUM(jl.credit), 0) != 0
		ORDER BY jl.third_party_nit`,
		companyID, accountCode, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("tercero balance: %w", err)
	}
	defer rows.Close()

	var out []*TerceroBalanceRow
	for rows.Next() {
		var r TerceroBalanceRow
		if err := rows.Scan(&r.TerceroNIT, &r.TotalDebit, &r.TotalCredit, &r.Balance); err != nil {
			return nil, fmt.Errorf("scan tercero balance: %w", err)
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// MediosMagneticos devuelve los movimientos por NIT y cuenta para un año completo.
// Es la base para generar el reporte de Información Exógena que se reporta a la DIAN.
// Solo incluye líneas que tengan third_party_nit registrado.
func (s *Service) MediosMagneticos(ctx context.Context, companyID uuid.UUID, year int) ([]*MediosMagneticosRow, error) {
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, 12, 31, 23, 59, 59, 999999999, time.UTC)

	rows, err := s.pool.Query(ctx, `
		SELECT
			jl.third_party_nit,
			a.code,
			a.name,
			a.category,
			COALESCE(SUM(jl.debit), 0)  AS total_debit,
			COALESCE(SUM(jl.credit), 0) AS total_credit
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		JOIN accounting.accounts a ON a.id = jl.account_id
		WHERE je.company_id  = $1
		  AND je.status      = 'POSTED'
		  AND je.date       >= $2
		  AND je.date       <= $3
		  AND jl.third_party_nit IS NOT NULL
		GROUP BY jl.third_party_nit, a.id, a.code, a.name, a.category
		ORDER BY jl.third_party_nit, a.code`,
		companyID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("medios magnéticos: %w", err)
	}
	defer rows.Close()

	var out []*MediosMagneticosRow
	for rows.Next() {
		var r MediosMagneticosRow
		if err := rows.Scan(&r.TerceroNIT, &r.AccountCode, &r.AccountName, &r.Category,
			&r.TotalDebit, &r.TotalCredit); err != nil {
			return nil, fmt.Errorf("scan medios magnéticos: %w", err)
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// CostCenterBalance devuelve el movimiento débito/crédito por centro de costo en un rango de fechas.
// Solo incluye líneas que tengan cost_center no nulo.
func (s *Service) CostCenterBalance(ctx context.Context, companyID uuid.UUID, from, to time.Time) (*CostCenterReport, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			jl.cost_center,
			COALESCE(SUM(jl.debit), 0)           AS total_debit,
			COALESCE(SUM(jl.credit), 0)           AS total_credit,
			COALESCE(SUM(jl.debit) - SUM(jl.credit), 0) AS balance
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		WHERE je.company_id = $1
		  AND je.status    = 'POSTED'
		  AND je.date     >= $2
		  AND je.date     <= $3
		  AND jl.cost_center IS NOT NULL
		GROUP BY jl.cost_center
		ORDER BY jl.cost_center`,
		companyID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("cost center balance: %w", err)
	}
	defer rows.Close()

	report := &CostCenterReport{}
	for rows.Next() {
		var r CostCenterRow
		if err := rows.Scan(&r.CostCenter, &r.TotalDebit, &r.TotalCredit, &r.Balance); err != nil {
			return nil, fmt.Errorf("scan cost center: %w", err)
		}
		if r.Balance > 0 {
			report.NetDebt += r.Balance
		}
		report.Rows = append(report.Rows, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return report, nil
}

// BalanceSheet devuelve el balance general a una fecha de corte.
// El parámetro opcional book ("PCGA" | "NIIF") filtra por libro contable; vacío = ambos libros.
func (s *Service) BalanceSheet(ctx context.Context, companyID uuid.UUID, asOf time.Time, book ...string) (*BalanceSheet, error) {
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
		  AND a.category IN ('Activo', 'Pasivo', 'Patrimonio')`+bookClause(resolveBook(book))+`
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
