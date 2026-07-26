package cartera

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository usando pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetMovements(ctx context.Context, companyID uuid.UUID, asOf time.Time, accountPrefixes []string) ([]*Movement, error) {
	likePatterns := makeLikePatterns(accountPrefixes)
	rows, err := r.pool.Query(ctx, `
		SELECT
			jl.id,
			je.id,
			je.date,
			COALESCE(je.description, ''),
			jl.third_party_nit,
			jl.debit,
			jl.credit,
			EXISTS(
				SELECT 1 FROM accounting.reconciliation_marks rm
				WHERE rm.journal_line_id = jl.id
			) AS reconciled
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		JOIN accounting.accounts a        ON a.id  = jl.account_id
		WHERE je.company_id         = $1
		  AND je.status             = 'POSTED'
		  AND je.date              <= $2
		  AND jl.third_party_nit   IS NOT NULL
		  AND a.code                LIKE ANY($3::text[])
		ORDER BY jl.third_party_nit, je.date, je.created_at`,
		companyID, asOf.UTC(), likePatterns,
	)
	if err != nil {
		return nil, fmt.Errorf("get movements: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *PostgresRepository) GetNITMovements(ctx context.Context, companyID uuid.UUID, nit string, accountPrefixes []string, from, to time.Time) ([]*Movement, error) {
	likePatterns := makeLikePatterns(accountPrefixes)
	rows, err := r.pool.Query(ctx, `
		SELECT
			jl.id,
			je.id,
			je.date,
			COALESCE(je.description, ''),
			jl.third_party_nit,
			jl.debit,
			jl.credit,
			EXISTS(
				SELECT 1 FROM accounting.reconciliation_marks rm
				WHERE rm.journal_line_id = jl.id
			) AS reconciled
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		JOIN accounting.accounts a        ON a.id  = jl.account_id
		WHERE je.company_id         = $1
		  AND je.status             = 'POSTED'
		  AND jl.third_party_nit    = $2
		  AND a.code                LIKE ANY($3::text[])
		  AND je.date              >= $4
		  AND je.date              <= $5
		ORDER BY je.date, je.created_at`,
		companyID, nit, likePatterns, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("get nit movements: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *PostgresRepository) MarkReconciled(ctx context.Context, mark ReconciliationMark) (*ReconciliationMark, error) {
	if mark.ID == uuid.Nil {
		mark.ID = uuid.New()
	}
	mark.ReconciledAt = time.Now().UTC()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO accounting.reconciliation_marks
			(id, company_id, journal_line_id, reconciled_with, note, reconciled_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		mark.ID, mark.CompanyID, mark.JournalLineID, mark.ReconciledWith,
		nullableString(mark.Note), mark.ReconciledAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrLineAlreadyReconciled
		}
		return nil, fmt.Errorf("mark reconciled: %w", err)
	}
	return &mark, nil
}

func (r *PostgresRepository) UnmarkReconciled(ctx context.Context, journalLineID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM accounting.reconciliation_marks
		WHERE journal_line_id = $1`, journalLineID)
	if err != nil {
		return fmt.Errorf("unmark reconciled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMarkNotFound
	}
	return nil
}

func scanMovements(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}) ([]*Movement, error) {
	var out []*Movement
	for rows.Next() {
		var m Movement
		var nit *string
		if err := rows.Scan(
			&m.LineID, &m.JournalID, &m.Date, &m.Description,
			&nit, &m.Debit, &m.Credit, &m.Reconciled,
		); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		if nit != nil {
			m.ThirdPartyNIT = *nit
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// makeLikePatterns convierte ["1305","1310"] en ["1305%","1310%"] para LIKE ANY.
func makeLikePatterns(prefixes []string) []string {
	out := make([]string, len(prefixes))
	for i, p := range prefixes {
		out[i] = p + "%"
	}
	return out
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
