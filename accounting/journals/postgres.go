package journals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository usando pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, entry JournalEntry) (*JournalEntry, error) {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	now := time.Now().UTC()
	entry.CreatedAt = now
	entry.UpdatedAt = now

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO accounting.journal_entries
			(id, company_id, period_id, date, description, status, source, entry_type,
			 voucher_type, voucher_number, source_document_id, source_document_type,
			 created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		entry.ID, entry.CompanyID, entry.PeriodID, entry.Date.UTC(),
		entry.Description, entry.Status, entry.Source, entry.EntryType,
		nullableString(entry.VoucherType), nullableString(entry.VoucherNumber),
		nullableUUID(entry.SourceDocumentID), nullableString(entry.SourceDocumentType),
		entry.CreatedAt, entry.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert journal entry: %w", err)
	}

	for i := range entry.Lines {
		line := entry.Lines[i]
		if line.ID == uuid.Nil {
			line.ID = uuid.New()
		}
		line.JournalID = entry.ID
		line.CreatedAt = now

		_, err = tx.Exec(ctx, `
			INSERT INTO accounting.journal_lines
				(id, journal_id, account_id, debit, credit, third_party_nit, cost_center, description, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			line.ID, line.JournalID, line.AccountID,
			line.Debit, line.Credit,
			nullableString(line.ThirdPartyNIT),
			nullableString(line.CostCenter), nullableString(line.Description),
			line.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert journal line: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit journal: %w", err)
	}
	return &entry, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*JournalEntry, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, company_id, period_id, date, description, status, source, entry_type,
		       voucher_type, voucher_number, source_document_id, source_document_type,
		       created_at, updated_at
		FROM accounting.journal_entries WHERE id = $1`, id)

	var e JournalEntry
	var voucherType, voucherNumber, sourceDocType *string
	var sourceDocID *uuid.UUID
	err := row.Scan(
		&e.ID, &e.CompanyID, &e.PeriodID, &e.Date,
		&e.Description, &e.Status, &e.Source, &e.EntryType,
		&voucherType, &voucherNumber, &sourceDocID, &sourceDocType,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrJournalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan journal entry: %w", err)
	}
	if voucherType != nil {
		e.VoucherType = *voucherType
	}
	if voucherNumber != nil {
		e.VoucherNumber = *voucherNumber
	}
	if sourceDocID != nil {
		e.SourceDocumentID = *sourceDocID
	}
	if sourceDocType != nil {
		e.SourceDocumentType = *sourceDocType
	}

	lines, err := r.loadLines(ctx, id)
	if err != nil {
		return nil, err
	}
	e.Lines = lines
	return &e, nil
}

func (r *PostgresRepository) loadLines(ctx context.Context, journalID uuid.UUID) ([]*JournalLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT jl.id, jl.journal_id, jl.account_id, a.code,
		       jl.debit, jl.credit, jl.third_party_nit, jl.cost_center, jl.description, jl.created_at
		FROM accounting.journal_lines jl
		JOIN accounting.accounts a ON a.id = jl.account_id
		WHERE jl.journal_id = $1
		ORDER BY jl.created_at`, journalID)
	if err != nil {
		return nil, fmt.Errorf("load lines: %w", err)
	}
	defer rows.Close()

	var lines []*JournalLine
	for rows.Next() {
		var l JournalLine
		var thirdPartyNIT, costCenter, description *string
		if err := rows.Scan(
			&l.ID, &l.JournalID, &l.AccountID, &l.AccountCode,
			&l.Debit, &l.Credit, &thirdPartyNIT, &costCenter, &description, &l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan line: %w", err)
		}
		if thirdPartyNIT != nil {
			l.ThirdPartyNIT = *thirdPartyNIT
		}
		if costCenter != nil {
			l.CostCenter = *costCenter
		}
		if description != nil {
			l.Description = *description
		}
		lines = append(lines, &l)
	}
	return lines, rows.Err()
}

func (r *PostgresRepository) Void(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounting.journal_entries
		SET status = $1, updated_at = $2
		WHERE id = $3 AND status = 'POSTED'`,
		StatusVoid, now, id,
	)
	if err != nil {
		return fmt.Errorf("void journal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJournalNotFound
	}
	return nil
}

func (r *PostgresRepository) ListByCompany(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*JournalEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, period_id, date, description, status, source, entry_type,
		       voucher_type, voucher_number, source_document_id, source_document_type,
		       created_at, updated_at
		FROM accounting.journal_entries
		WHERE company_id = $1
		ORDER BY date DESC, created_at DESC
		LIMIT $2 OFFSET $3`,
		companyID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list journals: %w", err)
	}
	defer rows.Close()

	var out []*JournalEntry
	for rows.Next() {
		var e JournalEntry
		var voucherType, voucherNumber, sourceDocType *string
		var sourceDocID *uuid.UUID
		if err := rows.Scan(
			&e.ID, &e.CompanyID, &e.PeriodID, &e.Date,
			&e.Description, &e.Status, &e.Source, &e.EntryType,
			&voucherType, &voucherNumber, &sourceDocID, &sourceDocType,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan journal: %w", err)
		}
		if voucherType != nil {
			e.VoucherType = *voucherType
		}
		if voucherNumber != nil {
			e.VoucherNumber = *voucherNumber
		}
		if sourceDocID != nil {
			e.SourceDocumentID = *sourceDocID
		}
		if sourceDocType != nil {
			e.SourceDocumentType = *sourceDocType
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetBySourceDocument(ctx context.Context, companyID, sourceDocID uuid.UUID, sourceDocType string) ([]*JournalEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, period_id, date, description, status, source, entry_type,
		       voucher_type, voucher_number, source_document_id, source_document_type,
		       created_at, updated_at
		FROM accounting.journal_entries
		WHERE company_id = $1
		  AND source_document_id = $2
		  AND source_document_type = $3
		ORDER BY date ASC, created_at ASC`,
		companyID, sourceDocID, sourceDocType,
	)
	if err != nil {
		return nil, fmt.Errorf("get by source document: %w", err)
	}
	defer rows.Close()

	var out []*JournalEntry
	for rows.Next() {
		var e JournalEntry
		var voucherType, voucherNumber, sourceDocTypeCol *string
		var sourceDocIDCol *uuid.UUID
		if err := rows.Scan(
			&e.ID, &e.CompanyID, &e.PeriodID, &e.Date,
			&e.Description, &e.Status, &e.Source, &e.EntryType,
			&voucherType, &voucherNumber, &sourceDocIDCol, &sourceDocTypeCol,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan journal by source doc: %w", err)
		}
		if voucherType != nil {
			e.VoucherType = *voucherType
		}
		if voucherNumber != nil {
			e.VoucherNumber = *voucherNumber
		}
		if sourceDocIDCol != nil {
			e.SourceDocumentID = *sourceDocIDCol
		}
		if sourceDocTypeCol != nil {
			e.SourceDocumentType = *sourceDocTypeCol
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetYearPLBalances(ctx context.Context, companyID uuid.UUID, year int) ([]PLBalance, error) {
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, 12, 31, 23, 59, 59, 999999999, time.UTC)

	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.code, a.name, a.category,
		       COALESCE(SUM(jl.debit) - SUM(jl.credit), 0) AS balance
		FROM accounting.accounts a
		JOIN accounting.journal_lines jl ON jl.account_id = a.id
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		WHERE je.company_id = $1
		  AND je.status     = 'POSTED'
		  AND je.date      >= $2
		  AND je.date      <= $3
		  AND a.category   IN ('Ingresos', 'Gastos', 'Costos', 'Costo de Producción')
		GROUP BY a.id, a.code, a.name, a.category
		HAVING COALESCE(SUM(jl.debit) - SUM(jl.credit), 0) != 0
		ORDER BY a.code`,
		companyID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("year pl balances: %w", err)
	}
	defer rows.Close()

	return scanPLBalances(rows)
}

func (r *PostgresRepository) GetBSBalances(ctx context.Context, companyID uuid.UUID, asOf time.Time) ([]PLBalance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.code, a.name, a.category,
		       COALESCE(SUM(jl.debit) - SUM(jl.credit), 0) AS balance
		FROM accounting.accounts a
		JOIN accounting.journal_lines jl ON jl.account_id = a.id
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		WHERE je.company_id = $1
		  AND je.status     = 'POSTED'
		  AND je.date      <= $2
		  AND a.category   IN ('Activo', 'Pasivo', 'Patrimonio')
		GROUP BY a.id, a.code, a.name, a.category
		HAVING COALESCE(SUM(jl.debit) - SUM(jl.credit), 0) != 0
		ORDER BY a.code`,
		companyID, asOf.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("bs balances: %w", err)
	}
	defer rows.Close()

	return scanPLBalances(rows)
}

// NextVoucherSeq incrementa atómicamente el contador y devuelve el nuevo valor.
// El upsert es atómico en PostgreSQL: safe para concurrencia sin locks manuales.
func (r *PostgresRepository) NextVoucherSeq(ctx context.Context, companyID uuid.UUID, code string, year int) (int, error) {
	var seq int
	err := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.voucher_counters (company_id, code, year, last_seq)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (company_id, code, year) DO UPDATE
			SET last_seq = voucher_counters.last_seq + 1
		RETURNING last_seq`,
		companyID, code, year,
	).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("next voucher seq %s/%d: %w", code, year, err)
	}
	return seq, nil
}

func (r *PostgresRepository) RegisterVoucherType(ctx context.Context, cfg VoucherTypeConfig) (*VoucherTypeConfig, error) {
	if cfg.ID == uuid.Nil {
		cfg.ID = uuid.New()
	}
	now := time.Now().UTC()
	cfg.CreatedAt = now
	cfg.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO accounting.voucher_types
			(id, company_id, code, name, resets_annually, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (company_id, code) DO UPDATE SET
			name            = EXCLUDED.name,
			resets_annually = EXCLUDED.resets_annually,
			is_active       = EXCLUDED.is_active,
			updated_at      = NOW()
		RETURNING id, created_at, updated_at`,
		cfg.ID, cfg.CompanyID, cfg.Code, cfg.Name,
		cfg.ResetsAnnually, cfg.IsActive, cfg.CreatedAt, cfg.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("register voucher type %s: %w", cfg.Code, err)
	}
	return &cfg, nil
}

func (r *PostgresRepository) ListVoucherTypes(ctx context.Context, companyID uuid.UUID) ([]*VoucherTypeConfig, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, code, name, resets_annually, is_active, created_at, updated_at
		FROM accounting.voucher_types
		WHERE company_id = $1 AND is_active = TRUE
		ORDER BY code`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list voucher types: %w", err)
	}
	defer rows.Close()

	var out []*VoucherTypeConfig
	for rows.Next() {
		var cfg VoucherTypeConfig
		if err := rows.Scan(
			&cfg.ID, &cfg.CompanyID, &cfg.Code, &cfg.Name,
			&cfg.ResetsAnnually, &cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan voucher type: %w", err)
		}
		out = append(out, &cfg)
	}
	return out, rows.Err()
}

func scanPLBalances(rows pgx.Rows) ([]PLBalance, error) {
	var out []PLBalance
	for rows.Next() {
		var b PLBalance
		if err := rows.Scan(&b.AccountID, &b.AccountCode, &b.AccountName, &b.Category, &b.Balance); err != nil {
			return nil, fmt.Errorf("scan pl balance: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
