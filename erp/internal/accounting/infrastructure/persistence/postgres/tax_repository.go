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

// ── IVA ──────────────────────────────────────────────────────────────────────────────────────

type IVADeclarationRepository struct{ pool *pgxpool.Pool }

func NewIVADeclarationRepository(pool *pgxpool.Pool) *IVADeclarationRepository {
	return &IVADeclarationRepository{pool: pool}
}

func (r *IVADeclarationRepository) Create(ctx context.Context, d domain.IVADeclaration) (*domain.IVADeclaration, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.iva_declarations
			(company_id, period_start, period_end, period_type, generated_iva, deductible_iva,
			 withheld_iva, net_iva, previous_balance, amount_to_pay, carry_forward, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'DRAFT')
		RETURNING id, company_id, period_start, period_end, period_type, generated_iva, deductible_iva,
		          withheld_iva, net_iva, previous_balance, amount_to_pay, carry_forward, status, journal_id, filed_at, created_at`,
		d.CompanyID, d.PeriodStart, d.PeriodEnd, string(d.PeriodType), d.GeneratedIVA, d.DeductibleIVA,
		d.WithheldIVA, d.NetIVA, d.PreviousBalance, d.AmountToPay, d.CarryForward,
	)
	return scanIVA(row)
}

func (r *IVADeclarationRepository) List(ctx context.Context, companyID uuid.UUID) ([]domain.IVADeclaration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, period_start, period_end, period_type, generated_iva, deductible_iva,
		       withheld_iva, net_iva, previous_balance, amount_to_pay, carry_forward, status, journal_id, filed_at, created_at
		FROM accounting.iva_declarations WHERE company_id=$1 ORDER BY period_start DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar declaraciones de IVA: %w", err)
	}
	defer rows.Close()

	var out []domain.IVADeclaration
	for rows.Next() {
		d, err := scanIVA(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *IVADeclarationRepository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.IVADeclaration, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, company_id, period_start, period_end, period_type, generated_iva, deductible_iva,
		       withheld_iva, net_iva, previous_balance, amount_to_pay, carry_forward, status, journal_id, filed_at, created_at
		FROM accounting.iva_declarations WHERE id=$1 AND company_id=$2`,
		id, companyID,
	)
	return scanIVA(row)
}

func (r *IVADeclarationRepository) GetLastCarryForward(ctx context.Context, companyID uuid.UUID, before time.Time) (int64, error) {
	var cf int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(carry_forward, 0) FROM accounting.iva_declarations
		WHERE company_id=$1 AND period_end < $2 ORDER BY period_end DESC LIMIT 1`,
		companyID, before,
	).Scan(&cf)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return cf, err
}

func (r *IVADeclarationRepository) MarkFiled(ctx context.Context, companyID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE accounting.iva_declarations SET status='FILED', filed_at=NOW() WHERE id=$1 AND company_id=$2`, id, companyID)
	return err
}

func scanIVA(row pgx.Row) (*domain.IVADeclaration, error) {
	var d domain.IVADeclaration
	var periodType, status string
	err := row.Scan(&d.ID, &d.CompanyID, &d.PeriodStart, &d.PeriodEnd, &periodType, &d.GeneratedIVA, &d.DeductibleIVA,
		&d.WithheldIVA, &d.NetIVA, &d.PreviousBalance, &d.AmountToPay, &d.CarryForward, &status, &d.JournalID, &d.FiledAt, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	d.PeriodType = domain.PeriodType(periodType)
	d.Status = domain.DeclarationStatus(status)
	return &d, nil
}

// ── Renta ────────────────────────────────────────────────────────────────────────────────────

type IncomeTaxDeclarationRepository struct{ pool *pgxpool.Pool }

func NewIncomeTaxDeclarationRepository(pool *pgxpool.Pool) *IncomeTaxDeclarationRepository {
	return &IncomeTaxDeclarationRepository{pool: pool}
}

func (r *IncomeTaxDeclarationRepository) Create(ctx context.Context, d domain.IncomeTaxDeclaration) (*domain.IncomeTaxDeclaration, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.income_tax_declarations
			(company_id, fiscal_year, taxable_income, tax_rate_bp, tax_computed, discounts,
			 tax_to_pay, advance_payments, amount_due, carry_forward, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'DRAFT')
		RETURNING id, company_id, fiscal_year, taxable_income, tax_rate_bp, tax_computed, discounts,
		          tax_to_pay, advance_payments, amount_due, carry_forward, status, journal_id, filed_at, created_at`,
		d.CompanyID, d.FiscalYear, d.TaxableIncome, d.TaxRateBP, d.TaxComputed, d.Discounts,
		d.TaxToPay, d.AdvancePayments, d.AmountDue, d.CarryForward,
	)
	return scanIncomeTax(row)
}

func (r *IncomeTaxDeclarationRepository) List(ctx context.Context, companyID uuid.UUID) ([]domain.IncomeTaxDeclaration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, fiscal_year, taxable_income, tax_rate_bp, tax_computed, discounts,
		       tax_to_pay, advance_payments, amount_due, carry_forward, status, journal_id, filed_at, created_at
		FROM accounting.income_tax_declarations WHERE company_id=$1 ORDER BY fiscal_year DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar declaraciones de renta: %w", err)
	}
	defer rows.Close()

	var out []domain.IncomeTaxDeclaration
	for rows.Next() {
		d, err := scanIncomeTax(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *IncomeTaxDeclarationRepository) GetRateForYear(ctx context.Context, year int) (int, error) {
	var rate int
	err := r.pool.QueryRow(ctx, "SELECT rate_bp FROM accounting.income_tax_rates WHERE year=$1", year).Scan(&rate)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("no hay tarifa de renta registrada para %d", year)
	}
	return rate, err
}

func (r *IncomeTaxDeclarationRepository) MarkFiled(ctx context.Context, companyID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE accounting.income_tax_declarations SET status='FILED', filed_at=NOW() WHERE id=$1 AND company_id=$2`, id, companyID)
	return err
}

func scanIncomeTax(row pgx.Row) (*domain.IncomeTaxDeclaration, error) {
	var d domain.IncomeTaxDeclaration
	var status string
	err := row.Scan(&d.ID, &d.CompanyID, &d.FiscalYear, &d.TaxableIncome, &d.TaxRateBP, &d.TaxComputed, &d.Discounts,
		&d.TaxToPay, &d.AdvancePayments, &d.AmountDue, &d.CarryForward, &status, &d.JournalID, &d.FiledAt, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	d.Status = domain.DeclarationStatus(status)
	return &d, nil
}

// ── ICA ──────────────────────────────────────────────────────────────────────────────────────

type ICATariffRepository struct{ pool *pgxpool.Pool }

func NewICATariffRepository(pool *pgxpool.Pool) *ICATariffRepository {
	return &ICATariffRepository{pool: pool}
}

func (r *ICATariffRepository) Create(ctx context.Context, t domain.ICATariff) (*domain.ICATariff, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.ica_tariffs (municipality_code, ciiu_code, fiscal_year, rate_bp, surcharge_bp)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (municipality_code, ciiu_code, fiscal_year) DO UPDATE
		SET rate_bp=EXCLUDED.rate_bp, surcharge_bp=EXCLUDED.surcharge_bp, updated_at=NOW()
		RETURNING id, municipality_code, ciiu_code, fiscal_year, rate_bp, surcharge_bp`,
		t.MunicipalityCode, t.CIIUCode, t.FiscalYear, t.RateBP, t.SurchargeBP,
	)
	return scanICATariff(row)
}

func (r *ICATariffRepository) List(ctx context.Context) ([]domain.ICATariff, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, municipality_code, ciiu_code, fiscal_year, rate_bp, surcharge_bp FROM accounting.ica_tariffs ORDER BY fiscal_year DESC, municipality_code")
	if err != nil {
		return nil, fmt.Errorf("listar tarifas ICA: %w", err)
	}
	defer rows.Close()

	var out []domain.ICATariff
	for rows.Next() {
		t, err := scanICATariff(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *ICATariffRepository) Get(ctx context.Context, municipalityCode, ciiuCode string, year int) (*domain.ICATariff, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, municipality_code, ciiu_code, fiscal_year, rate_bp, surcharge_bp
		FROM accounting.ica_tariffs WHERE municipality_code=$1 AND ciiu_code=$2 AND fiscal_year=$3`,
		municipalityCode, ciiuCode, year,
	)
	t, err := scanICATariff(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrICATariffNotFound
	}
	return t, err
}

func scanICATariff(row pgx.Row) (*domain.ICATariff, error) {
	var t domain.ICATariff
	err := row.Scan(&t.ID, &t.MunicipalityCode, &t.CIIUCode, &t.FiscalYear, &t.RateBP, &t.SurchargeBP)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type ICADeclarationRepository struct{ pool *pgxpool.Pool }

func NewICADeclarationRepository(pool *pgxpool.Pool) *ICADeclarationRepository {
	return &ICADeclarationRepository{pool: pool}
}

func (r *ICADeclarationRepository) Create(ctx context.Context, d domain.ICADeclaration) (*domain.ICADeclaration, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.ica_declarations
			(company_id, municipality_code, period_start, period_end, period_type, ciiu_code,
			 gross_revenue, deductions, net_base, tariff_bp, surcharge_bp, tax_computed,
			 surcharge_amount, tax_to_pay, previous_balance, amount_due, carry_forward, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,'DRAFT')
		RETURNING id, company_id, municipality_code, period_start, period_end, period_type, ciiu_code,
		          gross_revenue, deductions, net_base, tariff_bp, surcharge_bp, tax_computed,
		          surcharge_amount, tax_to_pay, previous_balance, amount_due, carry_forward, status, journal_id, filed_at, created_at`,
		d.CompanyID, d.MunicipalityCode, d.PeriodStart, d.PeriodEnd, string(d.PeriodType), d.CIIUCode,
		d.GrossRevenue, d.Deductions, d.NetBase, d.TariffBP, d.SurchargeBP, d.TaxComputed,
		d.SurchargeAmount, d.TaxToPay, d.PreviousBalance, d.AmountDue, d.CarryForward,
	)
	return scanICA(row)
}

func (r *ICADeclarationRepository) List(ctx context.Context, companyID uuid.UUID) ([]domain.ICADeclaration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, municipality_code, period_start, period_end, period_type, ciiu_code,
		       gross_revenue, deductions, net_base, tariff_bp, surcharge_bp, tax_computed,
		       surcharge_amount, tax_to_pay, previous_balance, amount_due, carry_forward, status, journal_id, filed_at, created_at
		FROM accounting.ica_declarations WHERE company_id=$1 ORDER BY period_start DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar declaraciones de ICA: %w", err)
	}
	defer rows.Close()

	var out []domain.ICADeclaration
	for rows.Next() {
		d, err := scanICA(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *ICADeclarationRepository) GetLastCarryForward(ctx context.Context, companyID uuid.UUID, municipalityCode string, before time.Time) (int64, error) {
	var cf int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(carry_forward, 0) FROM accounting.ica_declarations
		WHERE company_id=$1 AND municipality_code=$2 AND period_end < $3 ORDER BY period_end DESC LIMIT 1`,
		companyID, municipalityCode, before,
	).Scan(&cf)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return cf, err
}

func (r *ICADeclarationRepository) MarkFiled(ctx context.Context, companyID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE accounting.ica_declarations SET status='FILED', filed_at=NOW() WHERE id=$1 AND company_id=$2`, id, companyID)
	return err
}

func scanICA(row pgx.Row) (*domain.ICADeclaration, error) {
	var d domain.ICADeclaration
	var periodType, status string
	err := row.Scan(&d.ID, &d.CompanyID, &d.MunicipalityCode, &d.PeriodStart, &d.PeriodEnd, &periodType, &d.CIIUCode,
		&d.GrossRevenue, &d.Deductions, &d.NetBase, &d.TariffBP, &d.SurchargeBP, &d.TaxComputed,
		&d.SurchargeAmount, &d.TaxToPay, &d.PreviousBalance, &d.AmountDue, &d.CarryForward, &status, &d.JournalID, &d.FiledAt, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	d.PeriodType = domain.PeriodType(periodType)
	d.Status = domain.DeclarationStatus(status)
	return &d, nil
}
