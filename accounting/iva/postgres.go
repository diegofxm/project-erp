package iva

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

// QueryIVAMovements lee los movimientos de cuentas 2408xx para el período.
// Excluye los asientos con source = 'iva_payment' para evitar contar el pago
// de una declaración anterior como IVA descontable del período actual.
func (r *PostgresRepository) QueryIVAMovements(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]*IVALine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			a.code,
			a.name,
			COALESCE(SUM(jl.credit), 0) AS generated,
			COALESCE(SUM(jl.debit), 0)  AS deductible
		FROM accounting.accounts a
		JOIN accounting.journal_lines jl ON jl.account_id = a.id
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		WHERE je.company_id = $1
		  AND je.status     = 'POSTED'
		  AND je.date      >= $2
		  AND je.date      <= $3
		  AND a.code        LIKE '2408%'
		  AND je.source    != 'iva_payment'
		GROUP BY a.id, a.code, a.name
		HAVING COALESCE(SUM(jl.credit), 0) > 0 OR COALESCE(SUM(jl.debit), 0) > 0
		ORDER BY a.code`,
		companyID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("query iva movements: %w", err)
	}
	defer rows.Close()

	var out []*IVALine
	for rows.Next() {
		var l IVALine
		if err := rows.Scan(&l.AccountCode, &l.AccountName, &l.Generated, &l.Deductible); err != nil {
			return nil, fmt.Errorf("scan iva line: %w", err)
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

// QueryReteIVA lee los débitos en cuentas 1365xx para el período.
// Estos representan el IVA que clientes/agentes retuvieron a la empresa
// (reteiva a favor), que reduce el IVA neto a pagar.
func (r *PostgresRepository) QueryReteIVA(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]*ReteIVALine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			a.code,
			a.name,
			COALESCE(SUM(jl.debit), 0) AS withheld
		FROM accounting.accounts a
		JOIN accounting.journal_lines jl ON jl.account_id = a.id
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		WHERE je.company_id = $1
		  AND je.status     = 'POSTED'
		  AND je.date      >= $2
		  AND je.date      <= $3
		  AND a.code        LIKE '1365%'
		GROUP BY a.id, a.code, a.name
		HAVING COALESCE(SUM(jl.debit), 0) > 0
		ORDER BY a.code`,
		companyID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("query reteiva: %w", err)
	}
	defer rows.Close()

	var out []*ReteIVALine
	for rows.Next() {
		var l ReteIVALine
		if err := rows.Scan(&l.AccountCode, &l.AccountName, &l.Withheld); err != nil {
			return nil, fmt.Errorf("scan reteiva line: %w", err)
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Save(ctx context.Context, decl IVADeclaration) (*IVADeclaration, error) {
	if decl.ID == uuid.Nil {
		decl.ID = uuid.New()
	}
	now := time.Now().UTC()
	decl.CreatedAt = now
	decl.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO accounting.iva_declarations
			(id, company_id, period_start, period_end, period_type,
			 generated_iva, deductible_iva, withheld_iva, net_iva,
			 previous_balance, amount_to_pay, carry_forward,
			 status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (company_id, period_start, period_end) DO UPDATE SET
			generated_iva    = EXCLUDED.generated_iva,
			deductible_iva   = EXCLUDED.deductible_iva,
			withheld_iva     = EXCLUDED.withheld_iva,
			net_iva          = EXCLUDED.net_iva,
			previous_balance = EXCLUDED.previous_balance,
			amount_to_pay    = EXCLUDED.amount_to_pay,
			carry_forward    = EXCLUDED.carry_forward,
			updated_at       = NOW()`,
		decl.ID, decl.CompanyID, decl.PeriodStart.UTC(), decl.PeriodEnd.UTC(), string(decl.PeriodType),
		decl.GeneratedIVA, decl.DeductibleIVA, decl.WithheldIVA, decl.NetIVA,
		decl.PreviousBalance, decl.ToPay, decl.CarryForward,
		string(decl.Status), decl.CreatedAt, decl.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("save iva declaration: %w", err)
	}
	return &decl, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*IVADeclaration, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, company_id, period_start, period_end, period_type,
		       generated_iva, deductible_iva, withheld_iva, net_iva,
		       previous_balance, amount_to_pay, carry_forward,
		       status, journal_id, filed_at, created_at, updated_at
		FROM accounting.iva_declarations WHERE id = $1`, id)
	return scanDeclaration(row)
}

func (r *PostgresRepository) ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*IVADeclaration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, period_start, period_end, period_type,
		       generated_iva, deductible_iva, withheld_iva, net_iva,
		       previous_balance, amount_to_pay, carry_forward,
		       status, journal_id, filed_at, created_at, updated_at
		FROM accounting.iva_declarations
		WHERE company_id = $1
		ORDER BY period_start DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list iva declarations: %w", err)
	}
	defer rows.Close()

	var out []*IVADeclaration
	for rows.Next() {
		d, err := scanDeclaration(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status DeclarationStatus, journalID *uuid.UUID, filedAt *time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounting.iva_declarations
		SET status     = $1,
		    journal_id = COALESCE($2, journal_id),
		    filed_at   = COALESCE($3, filed_at),
		    updated_at = NOW()
		WHERE id = $4`,
		string(status), journalID, filedAt, id,
	)
	if err != nil {
		return fmt.Errorf("update iva declaration status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDeclarationNotFound
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDeclaration(row scannable) (*IVADeclaration, error) {
	var d IVADeclaration
	var journalID *uuid.UUID
	var filedAt *time.Time
	err := row.Scan(
		&d.ID, &d.CompanyID, &d.PeriodStart, &d.PeriodEnd, &d.PeriodType,
		&d.GeneratedIVA, &d.DeductibleIVA, &d.WithheldIVA, &d.NetIVA,
		&d.PreviousBalance, &d.ToPay, &d.CarryForward,
		&d.Status, &journalID, &filedAt,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeclarationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan iva declaration: %w", err)
	}
	if journalID != nil {
		d.JournalID = *journalID
	}
	d.FiledAt = filedAt
	return &d, nil
}
