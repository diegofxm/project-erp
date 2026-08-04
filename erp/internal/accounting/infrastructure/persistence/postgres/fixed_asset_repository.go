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

type FixedAssetRepository struct{ pool *pgxpool.Pool }

func NewFixedAssetRepository(pool *pgxpool.Pool) *FixedAssetRepository {
	return &FixedAssetRepository{pool: pool}
}

const fixedAssetCols = `id, company_id, code, name, COALESCE(description,''),
	asset_account, depreciation_account, accumulated_account, gain_account, loss_account,
	acquisition_date, acquisition_cost, salvage_value, useful_life_months, depreciation_method,
	status, COALESCE(third_party_nit,''), created_at, updated_at`

func (r *FixedAssetRepository) Create(ctx context.Context, a domain.FixedAsset) (*domain.FixedAsset, error) {
	// 424810 "Otros activos" (Utilidad en venta de otros bienes) y 531040 "Pérdidas por
	// siniestros" son los defaults genéricos reales más cercanos en el PUC extraído — no hay
	// una cuenta específica de "utilidad/pérdida en venta de PPE" posteable en ese catálogo.
	gain := a.GainAccount
	if gain == "" {
		gain = "424810"
	}
	loss := a.LossAccount
	if loss == "" {
		loss = "531040"
	}
	method := a.DepreciationMethod
	if method == "" {
		method = "STRAIGHT_LINE"
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.fixed_assets
			(company_id, code, name, description, asset_account, depreciation_account, accumulated_account,
			 gain_account, loss_account, acquisition_date, acquisition_cost, salvage_value, useful_life_months,
			 depreciation_method, status, third_party_nit)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'ACTIVE',$15)
		RETURNING `+fixedAssetCols,
		a.CompanyID, a.Code, a.Name, a.Description, a.AssetAccount, a.DepreciationAccount, a.AccumulatedAccount,
		gain, loss, a.AcquisitionDate, a.AcquisitionCost, a.SalvageValue, a.UsefulLifeMonths, method, a.ThirdPartyNIT,
	)
	return scanFixedAsset(row)
}

func (r *FixedAssetRepository) List(ctx context.Context, companyID uuid.UUID) ([]domain.FixedAsset, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+fixedAssetCols+" FROM accounting.fixed_assets WHERE company_id=$1 ORDER BY code", companyID)
	if err != nil {
		return nil, fmt.Errorf("listar activos fijos: %w", err)
	}
	defer rows.Close()

	var out []domain.FixedAsset
	for rows.Next() {
		a, err := scanFixedAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *FixedAssetRepository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.FixedAsset, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+fixedAssetCols+" FROM accounting.fixed_assets WHERE id=$1 AND company_id=$2", id, companyID)
	a, err := scanFixedAsset(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrFixedAssetNotFound
	}
	return a, err
}

func (r *FixedAssetRepository) GetAccumulatedDepreciation(ctx context.Context, assetID uuid.UUID) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, "SELECT COALESCE(SUM(amount),0) FROM accounting.depreciation_entries WHERE asset_id=$1", assetID).Scan(&total)
	return total, err
}

func (r *FixedAssetRepository) UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status domain.AssetStatus) error {
	_, err := r.pool.Exec(ctx, "UPDATE accounting.fixed_assets SET status=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3", string(status), id, companyID)
	return err
}

func scanFixedAsset(row pgx.Row) (*domain.FixedAsset, error) {
	var a domain.FixedAsset
	var status, method string
	err := row.Scan(&a.ID, &a.CompanyID, &a.Code, &a.Name, &a.Description,
		&a.AssetAccount, &a.DepreciationAccount, &a.AccumulatedAccount, &a.GainAccount, &a.LossAccount,
		&a.AcquisitionDate, &a.AcquisitionCost, &a.SalvageValue, &a.UsefulLifeMonths, &method,
		&status, &a.ThirdPartyNIT, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	a.Status = domain.AssetStatus(status)
	a.DepreciationMethod = method
	return &a, nil
}

// ── Corridas de depreciación ────────────────────────────────────────────────────────────────

type DepreciationRepository struct{ pool *pgxpool.Pool }

func NewDepreciationRepository(pool *pgxpool.Pool) *DepreciationRepository {
	return &DepreciationRepository{pool: pool}
}

func (r *DepreciationRepository) CreateRun(ctx context.Context, run domain.DepreciationRun, entries []domain.DepreciationEntry) (*domain.DepreciationRun, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	err = tx.QueryRow(ctx, `
		INSERT INTO accounting.depreciation_runs (company_id, period_id, run_date, status, journal_id)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at`,
		run.CompanyID, run.PeriodID, run.RunDate, string(run.Status), run.JournalID,
	).Scan(&run.ID, &run.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("crear corrida de depreciación: %w", err)
	}

	for _, e := range entries {
		_, err := tx.Exec(ctx, `
			INSERT INTO accounting.depreciation_entries (run_id, asset_id, amount)
			VALUES ($1,$2,$3)`,
			run.ID, e.AssetID, e.Amount,
		)
		if err != nil {
			return nil, fmt.Errorf("guardar cuota de depreciación: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *DepreciationRepository) ListRuns(ctx context.Context, companyID uuid.UUID) ([]domain.DepreciationRun, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, period_id, run_date, status, journal_id, created_at
		FROM accounting.depreciation_runs WHERE company_id=$1 ORDER BY run_date DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar corridas: %w", err)
	}
	defer rows.Close()

	var out []domain.DepreciationRun
	for rows.Next() {
		var run domain.DepreciationRun
		var status string
		if err := rows.Scan(&run.ID, &run.CompanyID, &run.PeriodID, &run.RunDate, &status, &run.JournalID, &run.CreatedAt); err != nil {
			return nil, err
		}
		run.Status = domain.DepreciationRunStatus(status)
		out = append(out, run)
	}
	return out, rows.Err()
}

func (r *DepreciationRepository) HasRunForPeriod(ctx context.Context, companyID, periodID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM accounting.depreciation_runs WHERE company_id=$1 AND period_id=$2 AND status='COMPLETED')`,
		companyID, periodID,
	).Scan(&exists)
	return exists, err
}
