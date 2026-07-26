package banking

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

// ── BankAccounts ──────────────────────────────────────────────────────────────

func (r *PostgresRepository) CreateBankAccount(ctx context.Context, ba BankAccount) (*BankAccount, error) {
	if ba.ID == uuid.Nil {
		ba.ID = uuid.New()
	}
	now := time.Now().UTC()
	ba.CreatedAt = now
	ba.UpdatedAt = now
	ba.IsActive = true

	_, err := r.pool.Exec(ctx, `
		INSERT INTO accounting.bank_accounts
			(id, company_id, name, bank_name, account_no, account_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		ba.ID, ba.CompanyID, ba.Name, ba.BankName, ba.AccountNo, ba.AccountID,
		ba.IsActive, ba.CreatedAt, ba.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrDuplicateAccountNo
		}
		return nil, fmt.Errorf("create bank account: %w", err)
	}
	return &ba, nil
}

func (r *PostgresRepository) GetBankAccount(ctx context.Context, id uuid.UUID) (*BankAccount, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, company_id, name, bank_name, account_no, account_id, is_active, created_at, updated_at
		FROM accounting.bank_accounts WHERE id = $1`, id)

	var ba BankAccount
	err := row.Scan(&ba.ID, &ba.CompanyID, &ba.Name, &ba.BankName, &ba.AccountNo,
		&ba.AccountID, &ba.IsActive, &ba.CreatedAt, &ba.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBankAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get bank account: %w", err)
	}
	return &ba, nil
}

func (r *PostgresRepository) ListBankAccounts(ctx context.Context, companyID uuid.UUID) ([]*BankAccount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, name, bank_name, account_no, account_id, is_active, created_at, updated_at
		FROM accounting.bank_accounts
		WHERE company_id = $1
		ORDER BY name`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list bank accounts: %w", err)
	}
	defer rows.Close()

	var out []*BankAccount
	for rows.Next() {
		var ba BankAccount
		if err := rows.Scan(&ba.ID, &ba.CompanyID, &ba.Name, &ba.BankName, &ba.AccountNo,
			&ba.AccountID, &ba.IsActive, &ba.CreatedAt, &ba.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan bank account: %w", err)
		}
		out = append(out, &ba)
	}
	return out, rows.Err()
}

// ── StatementLines ────────────────────────────────────────────────────────────

func (r *PostgresRepository) CreateStatementLines(ctx context.Context, lines []StatementLine) ([]*StatementLine, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	var out []*StatementLine
	for i := range lines {
		l := lines[i]
		if l.ID == uuid.Nil {
			l.ID = uuid.New()
		}
		l.CreatedAt = now
		l.IsReconciled = false

		_, err := tx.Exec(ctx, `
			INSERT INTO accounting.bank_statement_lines
				(id, bank_account_id, date, description, debit, credit, reference, is_reconciled, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			l.ID, l.BankAccountID, l.Date.UTC(), l.Description,
			l.Debit, l.Credit, nullableStr(l.Reference), l.IsReconciled, l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert statement line: %w", err)
		}
		out = append(out, &l)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit statement lines: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) GetStatementLine(ctx context.Context, id uuid.UUID) (*StatementLine, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, bank_account_id, date, description, debit, credit,
		       reference, is_reconciled, journal_line_id, created_at
		FROM accounting.bank_statement_lines WHERE id = $1`, id)
	return scanStatementLine(row)
}

func (r *PostgresRepository) ListStatementLines(ctx context.Context, bankAccountID uuid.UUID, from, to time.Time) ([]*StatementLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, bank_account_id, date, description, debit, credit,
		       reference, is_reconciled, journal_line_id, created_at
		FROM accounting.bank_statement_lines
		WHERE bank_account_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date, created_at`,
		bankAccountID, from.UTC(), to.UTC())
	if err != nil {
		return nil, fmt.Errorf("list statement lines: %w", err)
	}
	defer rows.Close()
	return scanStatementLines(rows)
}

func (r *PostgresRepository) ListUnreconciledStatement(ctx context.Context, bankAccountID uuid.UUID, from, to time.Time) ([]*StatementLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, bank_account_id, date, description, debit, credit,
		       reference, is_reconciled, journal_line_id, created_at
		FROM accounting.bank_statement_lines
		WHERE bank_account_id = $1 AND date >= $2 AND date <= $3 AND is_reconciled = FALSE
		ORDER BY date, created_at`,
		bankAccountID, from.UTC(), to.UTC())
	if err != nil {
		return nil, fmt.Errorf("list unreconciled statement: %w", err)
	}
	defer rows.Close()
	return scanStatementLines(rows)
}

func (r *PostgresRepository) Reconcile(ctx context.Context, statementLineID, journalLineID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounting.bank_statement_lines
		SET is_reconciled = TRUE, journal_line_id = $1
		WHERE id = $2 AND is_reconciled = FALSE`,
		journalLineID, statementLineID,
	)
	if err != nil {
		return fmt.Errorf("reconcile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadyReconciled
	}
	return nil
}

func (r *PostgresRepository) Unreconcile(ctx context.Context, statementLineID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounting.bank_statement_lines
		SET is_reconciled = FALSE, journal_line_id = NULL
		WHERE id = $1 AND is_reconciled = TRUE`,
		statementLineID,
	)
	if err != nil {
		return fmt.Errorf("unreconcile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotReconciled
	}
	return nil
}

func (r *PostgresRepository) ListUnreconciledBook(ctx context.Context, bankAccountID uuid.UUID, from, to time.Time) ([]*UnreconciledBookItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT jl.id, jl.journal_id, je.date, je.description, jl.debit, jl.credit
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		JOIN accounting.bank_accounts ba ON ba.account_id = jl.account_id
		WHERE ba.id = $1
		  AND je.status = 'POSTED'
		  AND je.date >= $2 AND je.date <= $3
		  AND NOT EXISTS (
		      SELECT 1 FROM accounting.bank_statement_lines bsl
		      WHERE bsl.journal_line_id = jl.id AND bsl.is_reconciled = TRUE
		  )
		ORDER BY je.date, je.created_at`,
		bankAccountID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("list unreconciled book: %w", err)
	}
	defer rows.Close()

	var out []*UnreconciledBookItem
	for rows.Next() {
		var item UnreconciledBookItem
		if err := rows.Scan(&item.JournalLineID, &item.JournalID, &item.Date,
			&item.Description, &item.Debit, &item.Credit); err != nil {
			return nil, fmt.Errorf("scan unreconciled book: %w", err)
		}
		out = append(out, &item)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) StatementBalance(ctx context.Context, bankAccountID uuid.UUID, asOf time.Time) (float64, error) {
	var balance float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(credit) - SUM(debit), 0)
		FROM accounting.bank_statement_lines
		WHERE bank_account_id = $1 AND date <= $2`,
		bankAccountID, asOf.UTC(),
	).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("statement balance: %w", err)
	}
	return balance, nil
}

func (r *PostgresRepository) BookBalance(ctx context.Context, bankAccountID uuid.UUID, asOf time.Time) (float64, error) {
	var balance float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(jl.debit) - SUM(jl.credit), 0)
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		JOIN accounting.bank_accounts ba ON ba.account_id = jl.account_id
		WHERE ba.id = $1 AND je.status = 'POSTED' AND je.date <= $2`,
		bankAccountID, asOf.UTC(),
	).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("book balance: %w", err)
	}
	return balance, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func scanStatementLine(row pgx.Row) (*StatementLine, error) {
	var l StatementLine
	var ref *string
	var journalLineID *uuid.UUID
	err := row.Scan(&l.ID, &l.BankAccountID, &l.Date, &l.Description,
		&l.Debit, &l.Credit, &ref, &l.IsReconciled, &journalLineID, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStatementLineNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan statement line: %w", err)
	}
	if ref != nil {
		l.Reference = *ref
	}
	l.JournalLineID = journalLineID
	return &l, nil
}

func scanStatementLines(rows pgx.Rows) ([]*StatementLine, error) {
	var out []*StatementLine
	for rows.Next() {
		l, err := scanStatementLine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "duplicate key") || contains(err.Error(), "unique constraint")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
