package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type JournalRepository struct {
	pool *pgxpool.Pool
}

func NewJournalRepository(pool *pgxpool.Pool) *JournalRepository {
	return &JournalRepository{pool: pool}
}

func (r *JournalRepository) Create(ctx context.Context, e domain.JournalEntry) (*domain.JournalEntry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const qEntry = `
		INSERT INTO accounting.journal_entries
			(company_id, period_id, date, description, status, source, entry_type,
			 voucher_type, voucher_number, source_document_id, source_document_type, book)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, company_id, period_id, date, description, status, source, entry_type,
		          voucher_type, voucher_number, source_document_id, source_document_type, book,
		          created_at, updated_at`

	status := e.Status
	if status == "" {
		status = domain.StatusPosted
	}
	source := e.Source
	if source == "" {
		source = "manual"
	}
	entryType := e.EntryType
	if entryType == "" {
		entryType = domain.EntryManual
	}
	book := e.Book
	if book == "" {
		book = domain.BookBoth
	}

	var voucherType, voucherNumber, sourceDocType *string
	var sourceDocID *uuid.UUID
	if e.VoucherType != "" {
		voucherType = &e.VoucherType
	}
	if e.VoucherNumber != "" {
		voucherNumber = &e.VoucherNumber
	}
	if e.SourceDocumentID != uuid.Nil {
		sourceDocID = &e.SourceDocumentID
	}
	if e.SourceDocumentType != "" {
		sourceDocType = &e.SourceDocumentType
	}

	row := tx.QueryRow(ctx, qEntry,
		e.CompanyID, e.PeriodID, e.Date, e.Description,
		status, source, entryType,
		voucherType, voucherNumber, sourceDocID, sourceDocType, book,
	)
	created, err := scanEntry(row)
	if err != nil {
		return nil, err
	}

	const qLine = `
		INSERT INTO accounting.journal_lines
			(journal_id, account_id, debit, credit, cost_center, description,
			 third_party_nit, foreign_amount, foreign_currency)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, journal_id, account_id, debit, credit,
		          COALESCE(cost_center,''), COALESCE(description,''),
		          COALESCE(third_party_nit,''), COALESCE(foreign_amount,0),
		          COALESCE(foreign_currency,''), created_at`

	for _, l := range e.Lines {
		var costCenter, desc, nit, foreignCur *string
		var foreignAmt *int64
		if l.CostCenter != "" {
			costCenter = &l.CostCenter
		}
		if l.Description != "" {
			desc = &l.Description
		}
		if l.ThirdPartyNIT != "" {
			nit = &l.ThirdPartyNIT
		}
		if l.ForeignAmount != 0 {
			foreignAmt = &l.ForeignAmount
		}
		if l.ForeignCurrency != "" {
			foreignCur = &l.ForeignCurrency
		}

		lRow := tx.QueryRow(ctx, qLine,
			created.ID, l.AccountID, l.Debit, l.Credit,
			costCenter, desc, nit, foreignAmt, foreignCur,
		)
		var line domain.JournalLine
		if err := lRow.Scan(
			&line.ID, &line.JournalID, &line.AccountID,
			&line.Debit, &line.Credit,
			&line.CostCenter, &line.Description,
			&line.ThirdPartyNIT, &line.ForeignAmount, &line.ForeignCurrency,
			&line.CreatedAt,
		); err != nil {
			return nil, err
		}
		line.AccountCode = l.AccountCode
		created.Lines = append(created.Lines, &line)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *JournalRepository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.JournalEntry, error) {
	const q = `
		SELECT id, company_id, period_id, date, description, status, source, entry_type,
		       voucher_type, voucher_number, source_document_id, source_document_type, book,
		       created_at, updated_at
		FROM accounting.journal_entries WHERE id = $1 AND company_id = $2`
	e, err := scanEntry(r.pool.QueryRow(ctx, q, id, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJournalNotFound
		}
		return nil, err
	}
	if err := r.loadLines(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (r *JournalRepository) Void(ctx context.Context, companyID, id uuid.UUID) error {
	const q = `
		UPDATE accounting.journal_entries
		SET status = 'VOID', updated_at = NOW() WHERE id = $1 AND company_id = $2`
	_, err := r.pool.Exec(ctx, q, id, companyID)
	return err
}

func (r *JournalRepository) ListByCompany(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*domain.JournalEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	const q = `
		SELECT id, company_id, period_id, date, description, status, source, entry_type,
		       voucher_type, voucher_number, source_document_id, source_document_type, book,
		       created_at, updated_at
		FROM accounting.journal_entries
		WHERE company_id = $1 ORDER BY date DESC, created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.JournalEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *JournalRepository) GetBySourceDocument(ctx context.Context, companyID, sourceDocID uuid.UUID, sourceDocType string) ([]*domain.JournalEntry, error) {
	const q = `
		SELECT id, company_id, period_id, date, description, status, source, entry_type,
		       voucher_type, voucher_number, source_document_id, source_document_type, book,
		       created_at, updated_at
		FROM accounting.journal_entries
		WHERE company_id = $1 AND source_document_id = $2 AND source_document_type = $3
		ORDER BY created_at`
	rows, err := r.pool.Query(ctx, q, companyID, sourceDocID, sourceDocType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.JournalEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// NextVoucherSeq incrementa atómicamente el contador para (companyID, code, year)
// y devuelve el nuevo valor. Usa INSERT … ON CONFLICT … UPDATE para ser atómico.
func (r *JournalRepository) NextVoucherSeq(ctx context.Context, companyID uuid.UUID, code string, year int) (int, error) {
	const q = `
		INSERT INTO accounting.voucher_counters (company_id, code, year, last_seq)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (company_id, code, year)
		DO UPDATE SET last_seq = accounting.voucher_counters.last_seq + 1
		RETURNING last_seq`
	var seq int
	if err := r.pool.QueryRow(ctx, q, companyID, code, year).Scan(&seq); err != nil {
		return 0, err
	}
	return seq, nil
}

func (r *JournalRepository) GetYearPLBalances(ctx context.Context, companyID uuid.UUID, year int) ([]domain.PLBalance, error) {
	const q = `
		SELECT a.id, a.code, a.name, a.category,
		       SUM(l.debit) - SUM(l.credit) AS balance
		FROM accounting.journal_lines l
		JOIN accounting.journal_entries e ON e.id = l.journal_id
		JOIN accounting.accounts a ON a.id = l.account_id
		WHERE e.company_id = $1
		  AND e.status = 'POSTED'
		  AND e.date BETWEEN make_date($2, 1, 1) AND make_date($2, 12, 31)
		  AND a.category IN ('Ingreso', 'Gasto', 'Costo', 'Costo de Producción')
		GROUP BY a.id, a.code, a.name, a.category
		HAVING SUM(l.debit) - SUM(l.credit) != 0
		ORDER BY a.code`
	return r.queryBalances(ctx, q, companyID, year)
}

func (r *JournalRepository) GetBSBalances(ctx context.Context, companyID uuid.UUID, asOf time.Time) ([]domain.PLBalance, error) {
	const q = `
		SELECT a.id, a.code, a.name, a.category,
		       SUM(l.debit) - SUM(l.credit) AS balance
		FROM accounting.journal_lines l
		JOIN accounting.journal_entries e ON e.id = l.journal_id
		JOIN accounting.accounts a ON a.id = l.account_id
		WHERE e.company_id = $1
		  AND e.status = 'POSTED'
		  AND e.date <= $2
		  AND a.category IN ('Activo', 'Pasivo', 'Patrimonio')
		GROUP BY a.id, a.code, a.name, a.category
		HAVING SUM(l.debit) - SUM(l.credit) != 0
		ORDER BY a.code`
	rows, err := r.pool.Query(ctx, q, companyID, asOf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBalances(rows)
}

// GetTrialBalance devuelve el movimiento (débito/crédito) y saldo de cada cuenta con
// actividad en el rango de fechas dado (Balance de Prueba).
func (r *JournalRepository) GetTrialBalance(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]domain.TrialBalanceRow, error) {
	const q = `
		SELECT a.id, a.code, a.name, a.category,
		       COALESCE(SUM(l.debit), 0) AS debit,
		       COALESCE(SUM(l.credit), 0) AS credit,
		       COALESCE(SUM(l.debit) - SUM(l.credit), 0) AS balance
		FROM accounting.journal_lines l
		JOIN accounting.journal_entries e ON e.id = l.journal_id
		JOIN accounting.accounts a ON a.id = l.account_id
		WHERE e.company_id = $1
		  AND e.status = 'POSTED'
		  AND e.date BETWEEN $2 AND $3
		GROUP BY a.id, a.code, a.name, a.category
		ORDER BY a.code`
	rows, err := r.pool.Query(ctx, q, companyID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TrialBalanceRow
	for rows.Next() {
		var row domain.TrialBalanceRow
		if err := rows.Scan(&row.AccountID, &row.AccountCode, &row.AccountName, &row.Category, &row.Debit, &row.Credit, &row.Balance); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *JournalRepository) GetIncomeInPeriod(ctx context.Context, companyID uuid.UUID, from, to time.Time) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.credit - l.debit), 0)
		FROM accounting.journal_lines l
		JOIN accounting.journal_entries e ON e.id = l.journal_id
		JOIN accounting.accounts a ON a.id = l.account_id
		WHERE e.company_id = $1 AND e.status = 'POSTED' AND a.category = 'Ingreso'
		  AND e.date BETWEEN $2 AND $3`,
		companyID, from, to,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("ingresos del período: %w", err)
	}
	return total, nil
}

// GetAccountLedger devuelve los movimientos de una cuenta en el rango dado, con saldo
// acumulado (Libro Mayor). El saldo corrido se calcula en Go recorriendo en orden cronológico.
func (r *JournalRepository) GetAccountLedger(ctx context.Context, companyID uuid.UUID, accountCode string, from, to time.Time) ([]domain.LedgerLine, error) {
	const q = `
		SELECT e.id, e.date, e.description, COALESCE(e.voucher_type,''), COALESCE(e.voucher_number,''),
		       l.debit, l.credit
		FROM accounting.journal_lines l
		JOIN accounting.journal_entries e ON e.id = l.journal_id
		JOIN accounting.accounts a ON a.id = l.account_id
		WHERE e.company_id = $1
		  AND e.status = 'POSTED'
		  AND a.code = $2
		  AND e.date BETWEEN $3 AND $4
		ORDER BY e.date, e.created_at`
	rows, err := r.pool.Query(ctx, q, companyID, accountCode, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LedgerLine
	var running int64
	for rows.Next() {
		var l domain.LedgerLine
		if err := rows.Scan(&l.JournalID, &l.Date, &l.Description, &l.VoucherType, &l.VoucherNumber, &l.Debit, &l.Credit); err != nil {
			return nil, err
		}
		running += l.Debit - l.Credit
		l.RunningBalance = running
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *JournalRepository) RegisterVoucherType(ctx context.Context, cfg domain.VoucherTypeConfig) (*domain.VoucherTypeConfig, error) {
	const q = `
		INSERT INTO accounting.voucher_types (company_id, code, name, resets_annually, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (company_id, code) DO UPDATE
		SET name = EXCLUDED.name, resets_annually = EXCLUDED.resets_annually,
		    is_active = EXCLUDED.is_active, updated_at = NOW()
		RETURNING id, company_id, code, name, resets_annually, is_active, created_at, updated_at`
	var out domain.VoucherTypeConfig
	err := r.pool.QueryRow(ctx, q,
		cfg.CompanyID, cfg.Code, cfg.Name, cfg.ResetsAnnually, cfg.IsActive,
	).Scan(
		&out.ID, &out.CompanyID, &out.Code, &out.Name,
		&out.ResetsAnnually, &out.IsActive, &out.CreatedAt, &out.UpdatedAt,
	)
	return &out, err
}

func (r *JournalRepository) IsRegisteredVoucherType(ctx context.Context, companyID uuid.UUID, code string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM accounting.voucher_types WHERE company_id = $1 AND code = $2 AND is_active)",
		companyID, code,
	).Scan(&exists)
	return exists, err
}

func (r *JournalRepository) ListVoucherTypes(ctx context.Context, companyID uuid.UUID) ([]*domain.VoucherTypeConfig, error) {
	const q = `
		SELECT id, company_id, code, name, resets_annually, is_active, created_at, updated_at
		FROM accounting.voucher_types WHERE company_id = $1 AND is_active ORDER BY code`
	rows, err := r.pool.Query(ctx, q, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.VoucherTypeConfig
	for rows.Next() {
		var c domain.VoucherTypeConfig
		if err := rows.Scan(
			&c.ID, &c.CompanyID, &c.Code, &c.Name,
			&c.ResetsAnnually, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *JournalRepository) loadLines(ctx context.Context, e *domain.JournalEntry) error {
	const q = `
		SELECT l.id, l.journal_id, l.account_id, a.code,
		       l.debit, l.credit,
		       COALESCE(l.cost_center,''), COALESCE(l.description,''),
		       COALESCE(l.third_party_nit,''), COALESCE(l.foreign_amount,0),
		       COALESCE(l.foreign_currency,''), l.created_at
		FROM accounting.journal_lines l
		JOIN accounting.accounts a ON a.id = l.account_id
		WHERE l.journal_id = $1 ORDER BY l.created_at`
	rows, err := r.pool.Query(ctx, q, e.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var l domain.JournalLine
		if err := rows.Scan(
			&l.ID, &l.JournalID, &l.AccountID, &l.AccountCode,
			&l.Debit, &l.Credit,
			&l.CostCenter, &l.Description,
			&l.ThirdPartyNIT, &l.ForeignAmount, &l.ForeignCurrency,
			&l.CreatedAt,
		); err != nil {
			return err
		}
		e.Lines = append(e.Lines, &l)
	}
	return rows.Err()
}

func (r *JournalRepository) queryBalances(ctx context.Context, q string, args ...any) ([]domain.PLBalance, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBalances(rows)
}

func scanBalances(rows pgx.Rows) ([]domain.PLBalance, error) {
	var out []domain.PLBalance
	for rows.Next() {
		var b domain.PLBalance
		if err := rows.Scan(&b.AccountID, &b.AccountCode, &b.AccountName, &b.Category, &b.Balance); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func scanEntry(row pgx.Row) (*domain.JournalEntry, error) {
	var e domain.JournalEntry
	var voucherType, voucherNumber, sourceDocType *string
	var sourceDocID *uuid.UUID
	err := row.Scan(
		&e.ID, &e.CompanyID, &e.PeriodID, &e.Date, &e.Description,
		&e.Status, &e.Source, &e.EntryType,
		&voucherType, &voucherNumber, &sourceDocID, &sourceDocType,
		&e.Book, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if voucherType != nil {
		e.VoucherType = *voucherType
	}
	if voucherNumber != nil {
		e.VoucherNumber = *voucherNumber
	}
	if sourceDocType != nil {
		e.SourceDocumentType = *sourceDocType
	}
	if sourceDocID != nil {
		e.SourceDocumentID = *sourceDocID
	}
	return &e, nil
}

func formatVoucherNumber(code string, year, seq int) string {
	return fmt.Sprintf("%s-%d-%05d", code, year, seq)
}
