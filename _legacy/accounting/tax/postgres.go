package tax

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/google/uuid"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository crea el repositorio PostgreSQL del módulo tax.
func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

// ── Helpers ──────────────────────────────────────────────────────────────────────────────────

func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

// ── F210 — Renta ─────────────────────────────────────────────────────────────────────────────

func (r *postgresRepository) GetIncomeTaxRate(ctx context.Context, year int) (*IncomeTaxRate, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT year, rate_bp, created_at FROM accounting.income_tax_rates WHERE year = $1`,
		year)
	var rate IncomeTaxRate
	if err := row.Scan(&rate.Year, &rate.RateBP, &rate.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRateNotFound
		}
		return nil, fmt.Errorf("get income tax rate %d: %w", year, err)
	}
	return &rate, nil
}

func (r *postgresRepository) SetIncomeTaxRate(ctx context.Context, year, rateBP int) (*IncomeTaxRate, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO accounting.income_tax_rates (year, rate_bp)
         VALUES ($1, $2)
         ON CONFLICT (year) DO UPDATE SET rate_bp = EXCLUDED.rate_bp
         RETURNING year, rate_bp, created_at`,
		year, rateBP)
	var rate IncomeTaxRate
	if err := row.Scan(&rate.Year, &rate.RateBP, &rate.CreatedAt); err != nil {
		return nil, fmt.Errorf("set income tax rate %d: %w", year, err)
	}
	return &rate, nil
}

func (r *postgresRepository) ListIncomeTaxRates(ctx context.Context) ([]*IncomeTaxRate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT year, rate_bp, created_at FROM accounting.income_tax_rates ORDER BY year DESC`)
	if err != nil {
		return nil, fmt.Errorf("list income tax rates: %w", err)
	}
	defer rows.Close()

	var rates []*IncomeTaxRate
	for rows.Next() {
		var rate IncomeTaxRate
		if err := rows.Scan(&rate.Year, &rate.RateBP, &rate.CreatedAt); err != nil {
			return nil, err
		}
		rates = append(rates, &rate)
	}
	return rates, rows.Err()
}

func (r *postgresRepository) SaveIncomeTaxDeclaration(ctx context.Context, d IncomeTaxDeclaration) (*IncomeTaxDeclaration, error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO accounting.income_tax_declarations
             (id, company_id, fiscal_year, taxable_income, tax_rate_bp, tax_computed,
              discounts, tax_to_pay, advance_payments, amount_due, carry_forward, status)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'DRAFT')
         ON CONFLICT (company_id, fiscal_year) DO UPDATE SET
             taxable_income   = EXCLUDED.taxable_income,
             tax_rate_bp      = EXCLUDED.tax_rate_bp,
             tax_computed     = EXCLUDED.tax_computed,
             discounts        = EXCLUDED.discounts,
             tax_to_pay       = EXCLUDED.tax_to_pay,
             advance_payments = EXCLUDED.advance_payments,
             amount_due       = EXCLUDED.amount_due,
             carry_forward    = EXCLUDED.carry_forward,
             updated_at       = NOW()
         RETURNING id, company_id, fiscal_year, taxable_income, tax_rate_bp, tax_computed,
                   discounts, tax_to_pay, advance_payments, amount_due, carry_forward,
                   status, journal_id, filed_at, created_at, updated_at`,
		d.ID, d.CompanyID, d.FiscalYear, d.TaxableIncome, d.TaxRateBP, d.TaxComputed,
		d.Discounts, d.TaxToPay, d.AdvancePayments, d.AmountDue, d.CarryForward,
	)
	return scanIncomeTaxDeclaration(row)
}

func (r *postgresRepository) GetIncomeTaxDeclarationByID(ctx context.Context, id uuid.UUID) (*IncomeTaxDeclaration, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, company_id, fiscal_year, taxable_income, tax_rate_bp, tax_computed,
                discounts, tax_to_pay, advance_payments, amount_due, carry_forward,
                status, journal_id, filed_at, created_at, updated_at
         FROM accounting.income_tax_declarations WHERE id = $1`, id)
	d, err := scanIncomeTaxDeclaration(row)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("income tax declaration %s: not found", id)
	}
	return d, err
}

func (r *postgresRepository) GetIncomeTaxDeclarationByYear(ctx context.Context, companyID uuid.UUID, year int) (*IncomeTaxDeclaration, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, company_id, fiscal_year, taxable_income, tax_rate_bp, tax_computed,
                discounts, tax_to_pay, advance_payments, amount_due, carry_forward,
                status, journal_id, filed_at, created_at, updated_at
         FROM accounting.income_tax_declarations
         WHERE company_id = $1 AND fiscal_year = $2`, companyID, year)
	d, err := scanIncomeTaxDeclaration(row)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("income tax declaration company=%s year=%d: not found", companyID, year)
	}
	return d, err
}

func (r *postgresRepository) ListIncomeTaxDeclarations(ctx context.Context, companyID uuid.UUID) ([]*IncomeTaxDeclaration, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, company_id, fiscal_year, taxable_income, tax_rate_bp, tax_computed,
                discounts, tax_to_pay, advance_payments, amount_due, carry_forward,
                status, journal_id, filed_at, created_at, updated_at
         FROM accounting.income_tax_declarations
         WHERE company_id = $1 ORDER BY fiscal_year DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list income tax declarations: %w", err)
	}
	defer rows.Close()

	var list []*IncomeTaxDeclaration
	for rows.Next() {
		d, err := scanIncomeTaxDeclaration(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *postgresRepository) UpdateIncomeTaxStatus(ctx context.Context, id uuid.UUID, status DeclarationStatus, journalID *uuid.UUID, filedAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE accounting.income_tax_declarations
         SET status = $2, journal_id = COALESCE($3, journal_id),
             filed_at = COALESCE($4, filed_at), updated_at = NOW()
         WHERE id = $1`,
		id, string(status), journalID, filedAt)
	return err
}

// scanIncomeTaxDeclaration es compatible con pgx.Row y pgx.Rows.
type incomeTaxScanner interface {
	Scan(dest ...any) error
}

func scanIncomeTaxDeclaration(row incomeTaxScanner) (*IncomeTaxDeclaration, error) {
	var d IncomeTaxDeclaration
	var journalID *uuid.UUID
	var filedAt *time.Time
	var status string
	err := row.Scan(
		&d.ID, &d.CompanyID, &d.FiscalYear, &d.TaxableIncome, &d.TaxRateBP, &d.TaxComputed,
		&d.Discounts, &d.TaxToPay, &d.AdvancePayments, &d.AmountDue, &d.CarryForward,
		&status, &journalID, &filedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	d.Status = DeclarationStatus(status)
	if journalID != nil {
		d.JournalID = *journalID
	}
	d.FiledAt = filedAt
	return &d, nil
}

// ── F220 — Certificados de Retención ─────────────────────────────────────────────────────────

func (r *postgresRepository) QueryWithholdingsByNIT(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]*WHByAccount, error) {
	// Agrega créditos a cuentas de retención del libro mayor por NIT tercero + account_code.
	// La query une con accounting.accounts para el nombre de la cuenta y con
	// accounting.withholding_concepts para el wh_type y rate_bp (primer concepto que comparta
	// account_payable). Solo considera asientos POSTED con third_party_nit poblado.
	rows, err := r.pool.Query(ctx, `
        SELECT
            jl.third_party_nit,
            jl.account_code,
            a.name                                                              AS account_name,
            COALESCE(
                (SELECT type FROM accounting.withholding_concepts
                 WHERE account_payable = jl.account_code AND is_active = TRUE LIMIT 1),
                'RETEFUENTE'
            )                                                                   AS wh_type,
            SUM(jl.credit)                                                      AS tax_withheld,
            COALESCE(
                (SELECT rate_bp FROM accounting.withholding_concepts
                 WHERE account_payable = jl.account_code AND is_active = TRUE LIMIT 1),
                0
            )                                                                   AS rate_bp
        FROM accounting.journal_lines jl
        JOIN accounting.journal_entries je ON je.id = jl.journal_id
        JOIN accounting.accounts a ON a.code = jl.account_code
        WHERE je.company_id = $1
          AND je.date BETWEEN $2 AND $3
          AND je.status = 'POSTED'
          AND jl.credit > 0
          AND jl.third_party_nit != ''
          AND jl.account_code IN (
              SELECT DISTINCT account_payable
              FROM accounting.withholding_concepts
              WHERE is_active = TRUE
          )
        GROUP BY jl.third_party_nit, jl.account_code, a.name
        HAVING SUM(jl.credit) > 0
        ORDER BY jl.third_party_nit, jl.account_code`,
		companyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query withholdings by NIT: %w", err)
	}
	defer rows.Close()

	var result []*WHByAccount
	for rows.Next() {
		var row WHByAccount
		if err := rows.Scan(&row.ThirdPartyNIT, &row.AccountCode, &row.AccountName,
			&row.WHType, &row.TaxWithheld, &row.RateBP); err != nil {
			return nil, err
		}
		result = append(result, &row)
	}
	return result, rows.Err()
}

func (r *postgresRepository) SaveCertificate(ctx context.Context, c WithholdingCertificate) (*WithholdingCertificate, error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO accounting.withholding_certificates
             (id, company_id, fiscal_year, third_party_nit, concept_code, concept_name,
              wh_type, gross_amount, tax_withheld, status)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'DRAFT')
         ON CONFLICT (company_id, fiscal_year, third_party_nit, concept_code, wh_type)
         DO UPDATE SET
             concept_name = EXCLUDED.concept_name,
             gross_amount = EXCLUDED.gross_amount,
             tax_withheld = EXCLUDED.tax_withheld,
             updated_at   = NOW()
         RETURNING id, company_id, fiscal_year, third_party_nit, concept_code, concept_name,
                   wh_type, gross_amount, tax_withheld, status, issued_at, created_at, updated_at`,
		c.ID, c.CompanyID, c.FiscalYear, c.ThirdPartyNIT, c.ConceptCode, c.ConceptName,
		c.WHType, c.GrossAmount, c.TaxWithheld,
	)
	return scanCertificate(row)
}

func (r *postgresRepository) GetCertificateByID(ctx context.Context, id uuid.UUID) (*WithholdingCertificate, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, company_id, fiscal_year, third_party_nit, concept_code, concept_name,
                wh_type, gross_amount, tax_withheld, status, issued_at, created_at, updated_at
         FROM accounting.withholding_certificates WHERE id = $1`, id)
	c, err := scanCertificate(row)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("withholding certificate %s: not found", id)
	}
	return c, err
}

func (r *postgresRepository) ListCertificates(ctx context.Context, companyID uuid.UUID, year int) ([]*WithholdingCertificate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, company_id, fiscal_year, third_party_nit, concept_code, concept_name,
                wh_type, gross_amount, tax_withheld, status, issued_at, created_at, updated_at
         FROM accounting.withholding_certificates
         WHERE company_id = $1 AND fiscal_year = $2
         ORDER BY third_party_nit, concept_code`, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("list certificates: %w", err)
	}
	defer rows.Close()

	var list []*WithholdingCertificate
	for rows.Next() {
		c, err := scanCertificate(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *postgresRepository) UpdateCertificateStatus(ctx context.Context, id uuid.UUID, status CertificateStatus, issuedAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE accounting.withholding_certificates
         SET status = $2, issued_at = COALESCE($3, issued_at), updated_at = NOW()
         WHERE id = $1`,
		id, string(status), issuedAt)
	return err
}

type certScanner interface {
	Scan(dest ...any) error
}

func scanCertificate(row certScanner) (*WithholdingCertificate, error) {
	var c WithholdingCertificate
	var status string
	err := row.Scan(
		&c.ID, &c.CompanyID, &c.FiscalYear, &c.ThirdPartyNIT, &c.ConceptCode, &c.ConceptName,
		&c.WHType, &c.GrossAmount, &c.TaxWithheld, &status, &c.IssuedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Status = CertificateStatus(status)
	return &c, nil
}

// ── F490 — ICA por Municipio ──────────────────────────────────────────────────────────────────

func (r *postgresRepository) SetIcaTariff(ctx context.Context, req IcaTariffRequest) (*IcaTariff, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO accounting.ica_tariffs
             (municipality_code, ciiu_code, fiscal_year, rate_bp, surcharge_bp)
         VALUES ($1,$2,$3,$4,$5)
         ON CONFLICT (municipality_code, ciiu_code, fiscal_year) DO UPDATE SET
             rate_bp      = EXCLUDED.rate_bp,
             surcharge_bp = EXCLUDED.surcharge_bp,
             updated_at   = NOW()
         RETURNING id, municipality_code, ciiu_code, fiscal_year, rate_bp, surcharge_bp, created_at, updated_at`,
		req.MunicipalityCode, req.CIIUCode, req.FiscalYear, req.RateBP, req.SurchargeBP)

	var t IcaTariff
	if err := row.Scan(&t.ID, &t.MunicipalityCode, &t.CIIUCode, &t.FiscalYear,
		&t.RateBP, &t.SurchargeBP, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, fmt.Errorf("set ica tariff: %w", err)
	}
	return &t, nil
}

func (r *postgresRepository) GetIcaTariff(ctx context.Context, municipalityCode, ciiuCode string, year int) (*IcaTariff, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, municipality_code, ciiu_code, fiscal_year, rate_bp, surcharge_bp, created_at, updated_at
         FROM accounting.ica_tariffs
         WHERE municipality_code = $1 AND ciiu_code = $2 AND fiscal_year = $3`,
		municipalityCode, ciiuCode, year)

	var t IcaTariff
	if err := row.Scan(&t.ID, &t.MunicipalityCode, &t.CIIUCode, &t.FiscalYear,
		&t.RateBP, &t.SurchargeBP, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTariffNotFound
		}
		return nil, fmt.Errorf("get ica tariff %s/%s/%d: %w", municipalityCode, ciiuCode, year, err)
	}
	return &t, nil
}

func (r *postgresRepository) SaveIcaDeclaration(ctx context.Context, d IcaDeclaration) (*IcaDeclaration, error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO accounting.ica_declarations
             (id, company_id, municipality_code, period_start, period_end, period_type,
              ciiu_code, gross_revenue, deductions, net_base, tariff_bp, surcharge_bp,
              tax_computed, surcharge_amount, tax_to_pay, previous_balance, amount_due,
              carry_forward, status)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,'DRAFT')
         ON CONFLICT (company_id, municipality_code, period_start, period_end, ciiu_code)
         DO UPDATE SET
             gross_revenue    = EXCLUDED.gross_revenue,
             deductions       = EXCLUDED.deductions,
             net_base         = EXCLUDED.net_base,
             tariff_bp        = EXCLUDED.tariff_bp,
             surcharge_bp     = EXCLUDED.surcharge_bp,
             tax_computed     = EXCLUDED.tax_computed,
             surcharge_amount = EXCLUDED.surcharge_amount,
             tax_to_pay       = EXCLUDED.tax_to_pay,
             previous_balance = EXCLUDED.previous_balance,
             amount_due       = EXCLUDED.amount_due,
             carry_forward    = EXCLUDED.carry_forward,
             updated_at       = NOW()
         RETURNING id, company_id, municipality_code, period_start, period_end, period_type,
                   ciiu_code, gross_revenue, deductions, net_base, tariff_bp, surcharge_bp,
                   tax_computed, surcharge_amount, tax_to_pay, previous_balance, amount_due,
                   carry_forward, status, journal_id, filed_at, created_at, updated_at`,
		d.ID, d.CompanyID, d.MunicipalityCode, d.PeriodStart, d.PeriodEnd, string(d.PeriodType),
		d.CIIUCode, d.GrossRevenue, d.Deductions, d.NetBase, d.TariffBP, d.SurchargeBP,
		d.TaxComputed, d.SurchargeAmount, d.TaxToPay, d.PreviousBalance, d.AmountDue, d.CarryForward,
	)
	return scanIcaDeclaration(row)
}

func (r *postgresRepository) GetIcaDeclarationByID(ctx context.Context, id uuid.UUID) (*IcaDeclaration, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, company_id, municipality_code, period_start, period_end, period_type,
                ciiu_code, gross_revenue, deductions, net_base, tariff_bp, surcharge_bp,
                tax_computed, surcharge_amount, tax_to_pay, previous_balance, amount_due,
                carry_forward, status, journal_id, filed_at, created_at, updated_at
         FROM accounting.ica_declarations WHERE id = $1`, id)
	d, err := scanIcaDeclaration(row)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("ica declaration %s: not found", id)
	}
	return d, err
}

func (r *postgresRepository) ListIcaDeclarations(ctx context.Context, companyID uuid.UUID) ([]*IcaDeclaration, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, company_id, municipality_code, period_start, period_end, period_type,
                ciiu_code, gross_revenue, deductions, net_base, tariff_bp, surcharge_bp,
                tax_computed, surcharge_amount, tax_to_pay, previous_balance, amount_due,
                carry_forward, status, journal_id, filed_at, created_at, updated_at
         FROM accounting.ica_declarations
         WHERE company_id = $1
         ORDER BY period_start DESC, municipality_code`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list ica declarations: %w", err)
	}
	defer rows.Close()

	var list []*IcaDeclaration
	for rows.Next() {
		d, err := scanIcaDeclaration(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *postgresRepository) UpdateIcaStatus(ctx context.Context, id uuid.UUID, status DeclarationStatus, journalID *uuid.UUID, filedAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE accounting.ica_declarations
         SET status = $2, journal_id = COALESCE($3, journal_id),
             filed_at = COALESCE($4, filed_at), updated_at = NOW()
         WHERE id = $1`,
		id, string(status), journalID, filedAt)
	return err
}

type icaScanner interface {
	Scan(dest ...any) error
}

func scanIcaDeclaration(row icaScanner) (*IcaDeclaration, error) {
	var d IcaDeclaration
	var status, periodType string
	var journalID *uuid.UUID
	err := row.Scan(
		&d.ID, &d.CompanyID, &d.MunicipalityCode, &d.PeriodStart, &d.PeriodEnd, &periodType,
		&d.CIIUCode, &d.GrossRevenue, &d.Deductions, &d.NetBase, &d.TariffBP, &d.SurchargeBP,
		&d.TaxComputed, &d.SurchargeAmount, &d.TaxToPay, &d.PreviousBalance, &d.AmountDue,
		&d.CarryForward, &status, &journalID, &d.FiledAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	d.Status = DeclarationStatus(status)
	d.PeriodType = IcaPeriodType(periodType)
	if journalID != nil {
		d.JournalID = *journalID
	}
	return &d, nil
}
