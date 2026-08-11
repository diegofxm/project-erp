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

type BankAccountRepository struct{ pool *pgxpool.Pool }

func NewBankAccountRepository(pool *pgxpool.Pool) *BankAccountRepository {
	return &BankAccountRepository{pool: pool}
}

func (r *BankAccountRepository) Create(ctx context.Context, a domain.BankAccount) (*domain.BankAccount, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.bank_accounts (company_id, name, bank_name, account_no, account_id, is_active)
		VALUES ($1,$2,$3,$4,$5,TRUE)
		RETURNING id, company_id, name, bank_name, account_no, account_id, is_active, created_at, updated_at`,
		a.CompanyID, a.Name, a.BankName, a.AccountNo, a.AccountID,
	)
	return scanBankAccount(row)
}

func (r *BankAccountRepository) List(ctx context.Context, companyID uuid.UUID) ([]domain.BankAccount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, name, bank_name, account_no, account_id, is_active, created_at, updated_at
		FROM accounting.bank_accounts WHERE company_id=$1 ORDER BY name`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar cuentas bancarias: %w", err)
	}
	defer rows.Close()

	var out []domain.BankAccount
	for rows.Next() {
		a, err := scanBankAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *BankAccountRepository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.BankAccount, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, company_id, name, bank_name, account_no, account_id, is_active, created_at, updated_at
		FROM accounting.bank_accounts WHERE id=$1 AND company_id=$2`,
		id, companyID,
	)
	a, err := scanBankAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBankAccountNotFound
	}
	return a, err
}

func (r *BankAccountRepository) Update(ctx context.Context, companyID, id uuid.UUID, name, bankName, accountNo string) (*domain.BankAccount, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE accounting.bank_accounts SET name=$3, bank_name=$4, account_no=$5, updated_at=NOW()
		WHERE id=$1 AND company_id=$2
		RETURNING id, company_id, name, bank_name, account_no, account_id, is_active, created_at, updated_at`,
		id, companyID, name, bankName, accountNo,
	)
	a, err := scanBankAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBankAccountNotFound
	}
	return a, err
}

func (r *BankAccountRepository) Deactivate(ctx context.Context, companyID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE accounting.bank_accounts SET is_active=FALSE, updated_at=NOW() WHERE id=$1 AND company_id=$2`,
		id, companyID,
	)
	if err != nil {
		return fmt.Errorf("desactivar cuenta bancaria: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrBankAccountNotFound
	}
	return nil
}

func scanBankAccount(row pgx.Row) (*domain.BankAccount, error) {
	var a domain.BankAccount
	err := row.Scan(&a.ID, &a.CompanyID, &a.Name, &a.BankName, &a.AccountNo, &a.AccountID, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ── Extractos ────────────────────────────────────────────────────────────────────────────────

type BankStatementRepository struct{ pool *pgxpool.Pool }

func NewBankStatementRepository(pool *pgxpool.Pool) *BankStatementRepository {
	return &BankStatementRepository{pool: pool}
}

func (r *BankStatementRepository) AddLine(ctx context.Context, l domain.BankStatementLine) (*domain.BankStatementLine, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.bank_statement_lines (bank_account_id, date, description, debit, credit, reference)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, bank_account_id, date, description, debit, credit, COALESCE(reference,''), is_reconciled, journal_line_id, created_at`,
		l.BankAccountID, l.Date, l.Description, l.Debit, l.Credit, l.Reference,
	)
	return scanStatementLine(row)
}

func (r *BankStatementRepository) ListByAccount(ctx context.Context, bankAccountID uuid.UUID) ([]domain.BankStatementLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, bank_account_id, date, description, debit, credit, COALESCE(reference,''), is_reconciled, journal_line_id, created_at
		FROM accounting.bank_statement_lines WHERE bank_account_id=$1 ORDER BY date DESC, created_at DESC`,
		bankAccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar extracto: %w", err)
	}
	defer rows.Close()

	var out []domain.BankStatementLine
	for rows.Next() {
		l, err := scanStatementLine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

func (r *BankStatementRepository) GetLine(ctx context.Context, id uuid.UUID) (*domain.BankStatementLine, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, bank_account_id, date, description, debit, credit, COALESCE(reference,''), is_reconciled, journal_line_id, created_at
		FROM accounting.bank_statement_lines WHERE id=$1`,
		id,
	)
	return scanStatementLine(row)
}

func (r *BankStatementRepository) Reconcile(ctx context.Context, id, journalLineID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounting.bank_statement_lines
		SET is_reconciled = TRUE, journal_line_id = $2
		WHERE id = $1 AND is_reconciled = FALSE`,
		id, journalLineID,
	)
	if err != nil {
		return fmt.Errorf("conciliar movimiento: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAlreadyReconciled
	}
	return nil
}

// FindCandidates busca líneas de asiento posteadas en la cuenta del banco, con el mismo monto
// (como débito o crédito), sin conciliar todavía, dentro de +/-15 días de la fecha dada.
func (r *BankStatementRepository) FindCandidates(ctx context.Context, companyID, accountID uuid.UUID, amountCents int64, around time.Time) ([]domain.ReconciliationCandidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.id, e.id, e.date, e.description, COALESCE(e.voucher_type,''), COALESCE(e.voucher_number,''), l.debit, l.credit
		FROM accounting.journal_lines l
		JOIN accounting.journal_entries e ON e.id = l.journal_id
		WHERE e.company_id = $1
		  AND e.status = 'POSTED'
		  AND l.account_id = $2
		  AND (l.debit = $3 OR l.credit = $3)
		  AND e.date BETWEEN $4::date - INTERVAL '15 days' AND $4::date + INTERVAL '15 days'
		  AND l.id NOT IN (SELECT journal_line_id FROM accounting.bank_statement_lines WHERE journal_line_id IS NOT NULL)
		ORDER BY e.date`,
		companyID, accountID, amountCents, around,
	)
	if err != nil {
		return nil, fmt.Errorf("buscar candidatos de conciliación: %w", err)
	}
	defer rows.Close()

	var out []domain.ReconciliationCandidate
	for rows.Next() {
		var c domain.ReconciliationCandidate
		if err := rows.Scan(&c.LineID, &c.JournalID, &c.Date, &c.Description, &c.VoucherType, &c.VoucherNumber, &c.Debit, &c.Credit); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanStatementLine(row pgx.Row) (*domain.BankStatementLine, error) {
	var l domain.BankStatementLine
	err := row.Scan(&l.ID, &l.BankAccountID, &l.Date, &l.Description, &l.Debit, &l.Credit,
		&l.Reference, &l.IsReconciled, &l.JournalLineID, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}
