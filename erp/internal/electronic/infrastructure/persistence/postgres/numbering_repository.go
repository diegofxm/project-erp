package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/electronic/domain"
	"github.com/diegofxm/erp/internal/shared/cryptutil"
)

type NumberingRepository struct {
	pool *pgxpool.Pool
	key  []byte
}

func NewNumberingRepository(pool *pgxpool.Pool, encryptionKey []byte) *NumberingRepository {
	return &NumberingRepository{pool: pool, key: encryptionKey}
}

const numberingCols = `id, company_id, dian_document_type_code, prefix, resolution_number,
	resolution_date, range_from, range_to, current_number, valid_from, valid_to, environment,
	technical_key, test_set_id, is_active, created_at, updated_at`

func (r *NumberingRepository) Create(ctx context.Context, nr domain.NumberingRange) (*domain.NumberingRange, error) {
	if nr.ID == uuid.Nil {
		nr.ID = uuid.New()
	}
	now := time.Now()
	nr.CreatedAt = now
	nr.UpdatedAt = now

	encKey, err := cryptutil.Encrypt(r.key, []byte(nr.TechnicalKey))
	if err != nil {
		return nil, fmt.Errorf("cifrar technical_key: %w", err)
	}

	var resNum *string
	if nr.ResolutionNumber != "" {
		resNum = &nr.ResolutionNumber
	}
	var resDate, vFrom, vTo *time.Time
	if !nr.ResolutionDate.IsZero() {
		resDate = &nr.ResolutionDate
	}
	if !nr.ValidFrom.IsZero() {
		vFrom = &nr.ValidFrom
	}
	if !nr.ValidTo.IsZero() {
		vTo = &nr.ValidTo
	}

	_, err = r.pool.Exec(ctx, `INSERT INTO electronic.numbering_ranges (`+numberingCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		nr.ID, nr.CompanyID, nr.DianDocumentTypeCode, nr.Prefix, resNum, resDate,
		nr.RangeFrom, nr.RangeTo, nr.CurrentNumber, vFrom, vTo, string(nr.Environment),
		encKey, nr.TestSetID, nr.IsActive, nr.CreatedAt, nr.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create numbering range: %w", err)
	}
	return &nr, nil
}

func (r *NumberingRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.NumberingRange, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+numberingCols+` FROM electronic.numbering_ranges WHERE id = $1`, id)
	return r.scan(row)
}

// ClaimNext incrementa atómicamente current_number y devuelve el nuevo valor.
func (r *NumberingRepository) ClaimNext(ctx context.Context, id uuid.UUID) (int64, error) {
	var claimed int64
	err := r.pool.QueryRow(ctx, `
		UPDATE electronic.numbering_ranges
		SET current_number = current_number + 1, updated_at = NOW()
		WHERE id = $1
		  AND is_active = TRUE
		  AND (range_to IS NULL OR current_number < range_to)
		RETURNING current_number`,
		id,
	).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.GetByID(ctx, id); getErr != nil {
			return 0, getErr
		}
		return 0, domain.ErrRangeExhausted
	}
	if err != nil {
		return 0, fmt.Errorf("claim next number: %w", err)
	}
	return claimed, nil
}

// ReleaseIfCurrent retrocede current_number si sigue siendo exactamente number.
func (r *NumberingRepository) ReleaseIfCurrent(ctx context.Context, id uuid.UUID, number int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE electronic.numbering_ranges
		SET current_number = current_number - 1, updated_at = NOW()
		WHERE id = $1 AND current_number = $2`,
		id, number,
	)
	return err
}

func (r *NumberingRepository) ClearTestSetID(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE electronic.numbering_ranges SET test_set_id = '', updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *NumberingRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE electronic.numbering_ranges SET is_active = FALSE, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrRangeNotFound
	}
	return nil
}

func (r *NumberingRepository) ListByCompany(ctx context.Context, companyID uuid.UUID, dianDocType string) ([]*domain.NumberingRange, error) {
	q := `SELECT ` + numberingCols + ` FROM electronic.numbering_ranges WHERE company_id = $1`
	args := []any{companyID}
	if dianDocType != "" {
		args = append(args, dianDocType)
		q += fmt.Sprintf(" AND dian_document_type_code = $%d", len(args))
	}
	q += " ORDER BY created_at DESC"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.NumberingRange
	for rows.Next() {
		nr, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, nr)
	}
	return out, rows.Err()
}

func (r *NumberingRepository) scan(row pgx.Row) (*domain.NumberingRange, error) {
	var nr domain.NumberingRange
	var env string
	var encKey []byte
	var resNum *string
	var resDate, vFrom, vTo *time.Time
	err := row.Scan(
		&nr.ID, &nr.CompanyID, &nr.DianDocumentTypeCode, &nr.Prefix, &resNum,
		&resDate, &nr.RangeFrom, &nr.RangeTo, &nr.CurrentNumber, &vFrom, &vTo,
		&env, &encKey, &nr.TestSetID, &nr.IsActive, &nr.CreatedAt, &nr.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrRangeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan numbering range: %w", err)
	}
	if resNum != nil {
		nr.ResolutionNumber = *resNum
	}
	if resDate != nil {
		nr.ResolutionDate = *resDate
	}
	if vFrom != nil {
		nr.ValidFrom = *vFrom
	}
	if vTo != nil {
		nr.ValidTo = *vTo
	}
	nr.Environment = domain.Environment(env)
	if len(encKey) > 0 {
		plain, err := cryptutil.Decrypt(r.key, encKey)
		if err != nil {
			return nil, fmt.Errorf("descifrar technical_key: %w", err)
		}
		nr.TechnicalKey = string(plain)
	}
	return &nr, nil
}
