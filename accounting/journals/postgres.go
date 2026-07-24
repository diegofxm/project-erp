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
			(id, company_id, period_id, date, description, status, source, entry_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		entry.ID, entry.CompanyID, entry.PeriodID, entry.Date.UTC(),
		entry.Description, entry.Status, entry.Source, entry.EntryType,
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
				(id, journal_id, account_id, debit, credit, cost_center, description, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			line.ID, line.JournalID, line.AccountID,
			line.Debit, line.Credit,
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
		SELECT id, company_id, period_id, date, description, status, source, entry_type, created_at, updated_at
		FROM accounting.journal_entries WHERE id = $1`, id)

	var e JournalEntry
	err := row.Scan(
		&e.ID, &e.CompanyID, &e.PeriodID, &e.Date,
		&e.Description, &e.Status, &e.Source, &e.EntryType,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrJournalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan journal entry: %w", err)
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
		       jl.debit, jl.credit, jl.cost_center, jl.description, jl.created_at
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
		var costCenter, description *string
		if err := rows.Scan(
			&l.ID, &l.JournalID, &l.AccountID, &l.AccountCode,
			&l.Debit, &l.Credit, &costCenter, &description, &l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan line: %w", err)
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
		SELECT id, company_id, period_id, date, description, status, source, entry_type, created_at, updated_at
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
		if err := rows.Scan(
			&e.ID, &e.CompanyID, &e.PeriodID, &e.Date,
			&e.Description, &e.Status, &e.Source, &e.EntryType,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan journal: %w", err)
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
