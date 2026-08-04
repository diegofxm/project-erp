package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type ReconciliationRepository struct {
	pool *pgxpool.Pool
}

func NewReconciliationRepository(pool *pgxpool.Pool) *ReconciliationRepository {
	return &ReconciliationRepository{pool: pool}
}

func (r *ReconciliationRepository) Mark(ctx context.Context, companyID, journalLineID uuid.UUID, reconciledWith *uuid.UUID, note string) (*domain.ReconciliationMark, error) {
	const q = `
		INSERT INTO accounting.reconciliation_marks (company_id, journal_line_id, reconciled_with, note)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (journal_line_id) DO UPDATE
		SET reconciled_with = EXCLUDED.reconciled_with, note = EXCLUDED.note, reconciled_at = NOW()
		RETURNING id, company_id, journal_line_id, reconciled_with, note, reconciled_at`
	var out domain.ReconciliationMark
	err := r.pool.QueryRow(ctx, q, companyID, journalLineID, reconciledWith, note).Scan(
		&out.ID, &out.CompanyID, &out.JournalLineID, &out.ReconciledWith, &out.Note, &out.ReconciledAt,
	)
	if err != nil {
		return nil, fmt.Errorf("marcar conciliación: %w", err)
	}
	return &out, nil
}

// Unmark borra la marca de journalLineID Y la del lado recíproco (la línea que la tenía como
// reconciled_with) — si solo se borrara un lado, la otra línea quedaría "atascada" mostrándose
// como conciliada aunque su contraparte ya no lo esté.
func (r *ReconciliationRepository) Unmark(ctx context.Context, companyID, journalLineID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"DELETE FROM accounting.reconciliation_marks WHERE company_id=$1 AND (journal_line_id=$2 OR reconciled_with=$2)",
		companyID, journalLineID,
	)
	if err != nil {
		return fmt.Errorf("quitar conciliación: %w", err)
	}
	return nil
}

func (r *ReconciliationRepository) ListOpenLines(ctx context.Context, companyID uuid.UUID, accountCode string) ([]domain.OpenLine, error) {
	const q = `
		SELECT l.id, e.id, e.date, e.description, COALESCE(e.voucher_number, ''),
		       COALESCE(l.third_party_nit, ''), l.debit, l.credit
		FROM accounting.journal_lines l
		JOIN accounting.journal_entries e ON e.id = l.journal_id
		JOIN accounting.accounts a ON a.id = l.account_id
		WHERE e.company_id = $1
		  AND e.status = 'POSTED'
		  AND a.code = $2
		  AND NOT EXISTS (
		      SELECT 1 FROM accounting.reconciliation_marks m WHERE m.journal_line_id = l.id
		  )
		ORDER BY e.date, e.created_at`
	rows, err := r.pool.Query(ctx, q, companyID, accountCode)
	if err != nil {
		return nil, fmt.Errorf("listar líneas sin conciliar: %w", err)
	}
	defer rows.Close()
	var out []domain.OpenLine
	for rows.Next() {
		var l domain.OpenLine
		if err := rows.Scan(&l.LineID, &l.JournalID, &l.Date, &l.Description, &l.VoucherNumber, &l.ThirdPartyNIT, &l.Debit, &l.Credit); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
