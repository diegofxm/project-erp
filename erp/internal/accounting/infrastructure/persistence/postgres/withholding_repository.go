package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type WithholdingConceptRepository struct{ pool *pgxpool.Pool }

func NewWithholdingConceptRepository(pool *pgxpool.Pool) *WithholdingConceptRepository {
	return &WithholdingConceptRepository{pool: pool}
}

const withholdingCols = "id, code, name, type, rate_bp, min_base_uvt, account_payable, account_receivable, applicable_to, is_active"

func (r *WithholdingConceptRepository) List(ctx context.Context) ([]domain.WithholdingConcept, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+withholdingCols+" FROM accounting.withholding_concepts WHERE is_active ORDER BY code, applicable_to")
	if err != nil {
		return nil, fmt.Errorf("listar conceptos de retención: %w", err)
	}
	defer rows.Close()

	var out []domain.WithholdingConcept
	for rows.Next() {
		c, err := scanWithholding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *WithholdingConceptRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WithholdingConcept, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+withholdingCols+" FROM accounting.withholding_concepts WHERE id=$1", id)
	c, err := scanWithholding(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWithholdingConceptNotFound
		}
		return nil, err
	}
	return c, nil
}

type WithholdingCertificateRepository struct{ pool *pgxpool.Pool }

func NewWithholdingCertificateRepository(pool *pgxpool.Pool) *WithholdingCertificateRepository {
	return &WithholdingCertificateRepository{pool: pool}
}

func (r *WithholdingCertificateRepository) Create(ctx context.Context, c domain.WithholdingCertificate) (*domain.WithholdingCertificate, error) {
	c.ID = uuid.New()
	err := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.withholding_certificates
			(id, company_id, fiscal_year, third_party_nit, concept_code, concept_name, wh_type,
			 gross_amount, tax_withheld, status, issued_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
		RETURNING issued_at, created_at`,
		c.ID, c.CompanyID, c.FiscalYear, c.ThirdPartyNIT, c.ConceptCode, c.ConceptName, c.WHType,
		c.GrossAmount, c.TaxWithheld, c.Status,
	).Scan(&c.IssuedAt, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("guardar certificado de retención: %w", err)
	}
	return &c, nil
}

func (r *WithholdingCertificateRepository) List(ctx context.Context, companyID uuid.UUID, year int) ([]domain.WithholdingCertificate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, fiscal_year, third_party_nit, concept_code, concept_name, wh_type,
		       gross_amount, tax_withheld, status, issued_at, created_at
		FROM accounting.withholding_certificates
		WHERE company_id = $1 AND fiscal_year = $2
		ORDER BY third_party_nit, concept_code`,
		companyID, year,
	)
	if err != nil {
		return nil, fmt.Errorf("listar certificados de retención: %w", err)
	}
	defer rows.Close()

	var out []domain.WithholdingCertificate
	for rows.Next() {
		var c domain.WithholdingCertificate
		if err := rows.Scan(&c.ID, &c.CompanyID, &c.FiscalYear, &c.ThirdPartyNIT, &c.ConceptCode, &c.ConceptName,
			&c.WHType, &c.GrossAmount, &c.TaxWithheld, &c.Status, &c.IssuedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanWithholding(row pgx.Row) (*domain.WithholdingConcept, error) {
	var c domain.WithholdingConcept
	err := row.Scan(&c.ID, &c.Code, &c.Name, &c.Type, &c.RateBP, &c.MinBaseUVT,
		&c.AccountPayable, &c.AccountReceivable, &c.ApplicableTo, &c.IsActive)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
