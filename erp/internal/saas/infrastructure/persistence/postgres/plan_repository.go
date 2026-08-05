package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type PlanRepository struct{ pool *pgxpool.Pool }

func NewPlanRepository(pool *pgxpool.Pool) *PlanRepository {
	return &PlanRepository{pool: pool}
}

const planCols = `id, code, name, description, billing_cycle, price_cents, included_documents,
	price_per_extra_document_cents, requires_certificate, certificate_price_cents,
	annual_increment_pct, is_internal, is_active, created_at, updated_at`

func scanPlan(row pgx.Row) (*domain.Plan, error) {
	var p domain.Plan
	var billingCycle string
	err := row.Scan(
		&p.ID, &p.Code, &p.Name, &p.Description, &billingCycle, &p.PriceCents, &p.IncludedDocuments,
		&p.PricePerExtraDocumentCents, &p.RequiresCertificate, &p.CertificatePriceCents,
		&p.AnnualIncrementPct, &p.IsInternal, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.BillingCycle = domain.BillingCycle(billingCycle)
	return &p, nil
}

func (r *PlanRepository) loadModuleCodes(ctx context.Context, planID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.code FROM saas.plan_modules pm
		JOIN saas.modules m ON m.id = pm.module_id
		WHERE pm.plan_id = $1
		ORDER BY m.code`, planID)
	if err != nil {
		return nil, fmt.Errorf("cargar módulos del plan: %w", err)
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		codes = append(codes, c)
	}
	return codes, rows.Err()
}

func (r *PlanRepository) setModules(ctx context.Context, tx pgx.Tx, planID uuid.UUID, moduleCodes []string) error {
	if _, err := tx.Exec(ctx, "DELETE FROM saas.plan_modules WHERE plan_id=$1", planID); err != nil {
		return fmt.Errorf("limpiar módulos del plan: %w", err)
	}
	for _, code := range moduleCodes {
		if _, err := tx.Exec(ctx, `
			INSERT INTO saas.plan_modules (plan_id, module_id)
			SELECT $1, id FROM saas.modules WHERE code = $2
			ON CONFLICT DO NOTHING`, planID, code,
		); err != nil {
			return fmt.Errorf("asociar módulo %q al plan: %w", code, err)
		}
	}
	return nil
}

func (r *PlanRepository) Create(ctx context.Context, p domain.Plan) (*domain.Plan, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("crear plan: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	p.ID = uuid.New()
	row := tx.QueryRow(ctx, `
		INSERT INTO saas.plans
			(id, code, name, description, billing_cycle, price_cents, included_documents,
			 price_per_extra_document_cents, requires_certificate, certificate_price_cents,
			 annual_increment_pct, is_internal, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING `+planCols,
		p.ID, p.Code, p.Name, p.Description, string(p.BillingCycle), p.PriceCents, p.IncludedDocuments,
		p.PricePerExtraDocumentCents, p.RequiresCertificate, p.CertificatePriceCents,
		p.AnnualIncrementPct, p.IsInternal, p.IsActive,
	)
	saved, err := scanPlan(row)
	if err != nil {
		return nil, fmt.Errorf("crear plan: %w", err)
	}
	if err := r.setModules(ctx, tx, saved.ID, p.ModuleCodes); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("crear plan: commit: %w", err)
	}
	saved.ModuleCodes = p.ModuleCodes
	return saved, nil
}

func (r *PlanRepository) Update(ctx context.Context, p domain.Plan) (*domain.Plan, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("actualizar plan: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	row := tx.QueryRow(ctx, `
		UPDATE saas.plans SET
			name = $2, description = $3, billing_cycle = $4, price_cents = $5,
			included_documents = $6, price_per_extra_document_cents = $7, requires_certificate = $8,
			certificate_price_cents = $9, annual_increment_pct = $10, is_active = $11,
			updated_at = NOW()
		WHERE id = $1
		RETURNING `+planCols,
		p.ID, p.Name, p.Description, string(p.BillingCycle), p.PriceCents, p.IncludedDocuments,
		p.PricePerExtraDocumentCents, p.RequiresCertificate, p.CertificatePriceCents,
		p.AnnualIncrementPct, p.IsActive,
	)
	saved, err := scanPlan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPlanNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("actualizar plan: %w", err)
	}
	if err := r.setModules(ctx, tx, saved.ID, p.ModuleCodes); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("actualizar plan: commit: %w", err)
	}
	saved.ModuleCodes = p.ModuleCodes
	return saved, nil
}

func (r *PlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+planCols+" FROM saas.plans WHERE id=$1", id)
	p, err := scanPlan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPlanNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener plan: %w", err)
	}
	codes, err := r.loadModuleCodes(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.ModuleCodes = codes
	return p, nil
}

func (r *PlanRepository) List(ctx context.Context) ([]domain.Plan, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+planCols+" FROM saas.plans ORDER BY price_cents")
	if err != nil {
		return nil, fmt.Errorf("listar planes: %w", err)
	}
	defer rows.Close()

	var out []domain.Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		codes, err := r.loadModuleCodes(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].ModuleCodes = codes
	}
	return out, nil
}
