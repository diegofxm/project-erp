package products

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diegofxm/apidian/internal/sqlutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository usando pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository crea el repositorio de productos sobre PostgreSQL.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const productColumns = `
	id,
	issuer_id,
	description,
	unit_code,
	unit_price_cents,
	item_code,
	item_type_code,
	item_type_name,
	item_type_agency_id,
	tax_type_code,
	tax_type_name,
	tax_percent,
	created_at,
	updated_at`

func (r *PostgresRepository) Create(ctx context.Context, p Product) (*Product, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	args := []any{
		p.ID, p.IssuerID, p.Description, p.UnitCode, p.UnitPriceCents,
		nullableString(p.ItemCode), nullableString(p.ItemTypeCode), nullableString(p.ItemTypeName), nullableString(p.ItemTypeAgencyID),
		nullableString(p.TaxTypeCode), nullableString(p.TaxTypeName), p.TaxPercent,
		p.CreatedAt, p.UpdatedAt,
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO edocuments.products (`+productColumns+`) VALUES (`+sqlutil.Placeholders(len(args))+`)`, args...)
	if err != nil {
		if isDuplicateItemCode(err) {
			return nil, ErrDuplicateItemCode
		}
		return nil, fmt.Errorf("create product: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+productColumns+` FROM edocuments.products WHERE id = $1`, id)
	return scanProduct(row)
}

func (r *PostgresRepository) ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]*Product, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+productColumns+` FROM edocuments.products WHERE issuer_id = $1 ORDER BY created_at DESC`, issuerID)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var out []*Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, issuerID, id uuid.UUID, p Product) (*Product, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE edocuments.products SET
			description = $1,
			unit_code = $2,
			unit_price_cents = $3,
			item_code = $4,
			item_type_code = $5,
			item_type_name = $6,
			item_type_agency_id = $7,
			tax_type_code = $8,
			tax_type_name = $9,
			tax_percent = $10,
			updated_at = $11
		WHERE id = $12 AND issuer_id = $13`,
		p.Description, p.UnitCode, p.UnitPriceCents,
		nullableString(p.ItemCode), nullableString(p.ItemTypeCode), nullableString(p.ItemTypeName), nullableString(p.ItemTypeAgencyID),
		nullableString(p.TaxTypeCode), nullableString(p.TaxTypeName), p.TaxPercent,
		time.Now().UTC(), id, issuerID,
	)
	if err != nil {
		if isDuplicateItemCode(err) {
			return nil, ErrDuplicateItemCode
		}
		return nil, fmt.Errorf("update product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrProductNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *PostgresRepository) Delete(ctx context.Context, issuerID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM edocuments.products WHERE id = $1 AND issuer_id = $2`, id, issuerID)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func scanProduct(row pgx.Row) (*Product, error) {
	var p Product
	var itemCode, itemTypeCode, itemTypeName, itemTypeAgencyID, taxTypeCode, taxTypeName *string

	err := row.Scan(
		&p.ID, &p.IssuerID, &p.Description, &p.UnitCode, &p.UnitPriceCents,
		&itemCode, &itemTypeCode, &itemTypeName, &itemTypeAgencyID,
		&taxTypeCode, &taxTypeName, &p.TaxPercent,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan product: %w", err)
	}

	p.ItemCode = deref(itemCode)
	p.ItemTypeCode = deref(itemTypeCode)
	p.ItemTypeName = deref(itemTypeName)
	p.ItemTypeAgencyID = deref(itemTypeAgencyID)
	p.TaxTypeCode = deref(taxTypeCode)
	p.TaxTypeName = deref(taxTypeName)
	return &p, nil
}

// isDuplicateItemCode detecta la violación del índice único parcial (issuer_id, item_code) —
// ver migración 000011_products_item_code_unique. Mismo patrón que issuers/postgres.go
// (isDuplicateKey).
func isDuplicateItemCode(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
