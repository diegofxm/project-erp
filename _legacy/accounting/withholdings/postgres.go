package withholdings

import (
	"context"
	"errors"
	"fmt"

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

func (r *PostgresRepository) GetConcept(ctx context.Context, code string, wType WithholdingType, vendorType VendorType) (*Concept, error) {
	// Prioriza la fila con applicable_to exacto (NATURAL/JURIDICA) sobre BOTH.
	row := r.pool.QueryRow(ctx, `
		SELECT id, code, name, type, rate_bp, min_base_uvt,
		       account_payable, account_receivable, applicable_to, is_active,
		       created_at, updated_at
		FROM accounting.withholding_concepts
		WHERE code          = $1
		  AND type          = $2
		  AND (applicable_to = $3 OR applicable_to = 'BOTH')
		  AND is_active     = TRUE
		ORDER BY CASE WHEN applicable_to = $3 THEN 0 ELSE 1 END
		LIMIT 1`,
		code, string(wType), string(vendorType),
	)
	return scanConcept(row)
}

func (r *PostgresRepository) ListConcepts(ctx context.Context, wType WithholdingType) ([]*Concept, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, type, rate_bp, min_base_uvt,
		       account_payable, account_receivable, applicable_to, is_active,
		       created_at, updated_at
		FROM accounting.withholding_concepts
		WHERE type      = $1
		  AND is_active = TRUE
		ORDER BY code, applicable_to`,
		string(wType),
	)
	if err != nil {
		return nil, fmt.Errorf("list withholding concepts: %w", err)
	}
	defer rows.Close()

	var out []*Concept
	for rows.Next() {
		c, err := scanConcept(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetUVT(ctx context.Context, year int) (*UVTValue, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT year, value_cents FROM accounting.uvt_values WHERE year = $1`, year)
	var u UVTValue
	if err := row.Scan(&u.Year, &u.ValueCents); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %d", ErrUVTNotFound, year)
		}
		return nil, fmt.Errorf("get uvt %d: %w", year, err)
	}
	return &u, nil
}

func (r *PostgresRepository) ListUVT(ctx context.Context) ([]*UVTValue, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT year, value_cents FROM accounting.uvt_values ORDER BY year DESC`)
	if err != nil {
		return nil, fmt.Errorf("list uvt: %w", err)
	}
	defer rows.Close()

	var out []*UVTValue
	for rows.Next() {
		var u UVTValue
		if err := rows.Scan(&u.Year, &u.ValueCents); err != nil {
			return nil, fmt.Errorf("scan uvt: %w", err)
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

// scanConcept lee una fila usando la interfaz compartida por QueryRow y rows.Next().
type scannable interface {
	Scan(dest ...any) error
}

func scanConcept(row scannable) (*Concept, error) {
	var c Concept
	err := row.Scan(
		&c.ID, &c.Code, &c.Name, &c.Type, &c.RateBP, &c.MinBaseUVT,
		&c.AccountPayable, &c.AccountReceivable, &c.ApplicableTo, &c.IsActive,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConceptNotFound
		}
		return nil, fmt.Errorf("scan concept: %w", err)
	}
	return &c, nil
}
