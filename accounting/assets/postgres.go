package assets

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

func (r *PostgresRepository) Create(ctx context.Context, asset FixedAsset) (*FixedAsset, error) {
	if asset.ID == uuid.Nil {
		asset.ID = uuid.New()
	}
	now := time.Now().UTC()
	asset.CreatedAt = now
	asset.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO accounting.fixed_assets
			(id, company_id, code, name, description,
			 asset_account, depreciation_account, accumulated_account,
			 gain_account, loss_account,
			 acquisition_date, acquisition_cost, salvage_value,
			 useful_life_months, depreciation_method, status,
			 third_party_nit, source_document_id, source_document_type,
			 created_at, updated_at)
		VALUES
			($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		asset.ID, asset.CompanyID, asset.Code, asset.Name,
		nullableString(asset.Description),
		asset.AssetAccount, asset.DepreciationAccount, asset.AccumulatedAccount,
		asset.GainAccount, asset.LossAccount,
		asset.AcquisitionDate.UTC(), asset.AcquisitionCost, asset.SalvageValue,
		asset.UsefulLifeMonths, string(asset.DepreciationMethod), string(asset.Status),
		nullableString(asset.ThirdPartyNIT),
		nullableUUID(asset.SourceDocumentID), nullableString(asset.SourceDocumentType),
		asset.CreatedAt, asset.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create fixed asset: %w", err)
	}
	return &asset, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*FixedAsset, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, company_id, code, name, description,
		       asset_account, depreciation_account, accumulated_account,
		       gain_account, loss_account,
		       acquisition_date, acquisition_cost, salvage_value,
		       useful_life_months, depreciation_method, status,
		       third_party_nit, source_document_id, source_document_type,
		       created_at, updated_at
		FROM accounting.fixed_assets WHERE id = $1`, id)
	return scanAsset(row)
}

func (r *PostgresRepository) ListByCompany(ctx context.Context, companyID uuid.UUID, activeOnly bool) ([]*FixedAsset, error) {
	query := `
		SELECT id, company_id, code, name, description,
		       asset_account, depreciation_account, accumulated_account,
		       gain_account, loss_account,
		       acquisition_date, acquisition_cost, salvage_value,
		       useful_life_months, depreciation_method, status,
		       third_party_nit, source_document_id, source_document_type,
		       created_at, updated_at
		FROM accounting.fixed_assets
		WHERE company_id = $1`
	if activeOnly {
		query += " AND status = 'ACTIVE'"
	}
	query += " ORDER BY code"

	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("list fixed assets: %w", err)
	}
	defer rows.Close()

	var out []*FixedAsset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status AssetStatus) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounting.fixed_assets
		SET status = $1, updated_at = NOW()
		WHERE id = $2`,
		string(status), id,
	)
	if err != nil {
		return fmt.Errorf("update asset status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAssetNotFound
	}
	return nil
}

func (r *PostgresRepository) GetAccumulatedDepreciation(ctx context.Context, assetID uuid.UUID) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(de.amount), 0)
		FROM accounting.depreciation_entries de
		JOIN accounting.depreciation_runs dr ON dr.id = de.run_id
		WHERE de.asset_id = $1 AND dr.status = 'COMPLETED'`, assetID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get accumulated depreciation: %w", err)
	}
	return total, nil
}

func (r *PostgresRepository) GetRunForPeriod(ctx context.Context, companyID, periodID uuid.UUID) (*DepreciationRun, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, company_id, period_id, run_date, status, journal_id, created_at
		FROM accounting.depreciation_runs
		WHERE company_id = $1 AND period_id = $2 AND status = 'COMPLETED'`,
		companyID, periodID,
	)
	var run DepreciationRun
	var journalID *uuid.UUID
	err := row.Scan(&run.ID, &run.CompanyID, &run.PeriodID, &run.RunDate,
		&run.Status, &journalID, &run.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // no existe — es válido correr la depreciación
	}
	if err != nil {
		return nil, fmt.Errorf("get run for period: %w", err)
	}
	if journalID != nil {
		run.JournalID = *journalID
	}
	return &run, nil
}

func (r *PostgresRepository) CreateRun(ctx context.Context, run DepreciationRun, entries []DepreciationEntry) (*DepreciationRun, error) {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	now := time.Now().UTC()
	run.CreatedAt = now

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var journalID *uuid.UUID
	if run.JournalID != uuid.Nil {
		journalID = &run.JournalID
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO accounting.depreciation_runs
			(id, company_id, period_id, run_date, status, journal_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		run.ID, run.CompanyID, run.PeriodID,
		run.RunDate.UTC(), run.Status, journalID, run.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert depreciation run: %w", err)
	}

	for i := range entries {
		e := &entries[i]
		if e.ID == uuid.Nil {
			e.ID = uuid.New()
		}
		e.RunID = run.ID
		e.CreatedAt = now

		_, err = tx.Exec(ctx, `
			INSERT INTO accounting.depreciation_entries (id, run_id, asset_id, amount, created_at)
			VALUES ($1,$2,$3,$4,$5)`,
			e.ID, e.RunID, e.AssetID, e.Amount, e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert depreciation entry: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit depreciation run: %w", err)
	}
	return &run, nil
}

// scannable es una interfaz compartida entre pgx.Row y pgx.Rows.
type scannable interface {
	Scan(dest ...any) error
}

func scanAsset(row scannable) (*FixedAsset, error) {
	var a FixedAsset
	var description, thirdPartyNIT, sourceDocType *string
	var sourceDocID *uuid.UUID
	err := row.Scan(
		&a.ID, &a.CompanyID, &a.Code, &a.Name, &description,
		&a.AssetAccount, &a.DepreciationAccount, &a.AccumulatedAccount,
		&a.GainAccount, &a.LossAccount,
		&a.AcquisitionDate, &a.AcquisitionCost, &a.SalvageValue,
		&a.UsefulLifeMonths, &a.DepreciationMethod, &a.Status,
		&thirdPartyNIT, &sourceDocID, &sourceDocType,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAssetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan fixed asset: %w", err)
	}
	if description != nil {
		a.Description = *description
	}
	if thirdPartyNIT != nil {
		a.ThirdPartyNIT = *thirdPartyNIT
	}
	if sourceDocID != nil {
		a.SourceDocumentID = *sourceDocID
	}
	if sourceDocType != nil {
		a.SourceDocumentType = *sourceDocType
	}
	return &a, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
