package forex

import (
	"context"
	"errors"
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

func (r *PostgresRepository) SetRate(ctx context.Context, req SetRateRequest) (*ExchangeRate, error) {
	if req.RateX10000 <= 0 {
		return nil, ErrInvalidRate
	}
	source := req.Source
	if source == "" {
		source = "MANUAL"
	}
	var er ExchangeRate
	err := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.exchange_rates (rate_date, from_currency, to_currency, rate_x10000, source)
		VALUES ($1, $2, 'COP', $3, $4)
		ON CONFLICT (rate_date, from_currency, to_currency)
		DO UPDATE SET rate_x10000 = EXCLUDED.rate_x10000, source = EXCLUDED.source
		RETURNING id, rate_date, from_currency, to_currency, rate_x10000, source, created_at`,
		req.Date.Format("2006-01-02"), req.FromCurrency, req.RateX10000, source,
	).Scan(&er.ID, &er.Date, &er.FromCurrency, &er.ToCurrency, &er.RateX10000, &er.Source, &er.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &er, nil
}

func (r *PostgresRepository) GetRate(ctx context.Context, date time.Time, fromCurrency string) (*ExchangeRate, error) {
	var er ExchangeRate
	err := r.pool.QueryRow(ctx, `
		SELECT id, rate_date, from_currency, to_currency, rate_x10000, source, created_at
		FROM accounting.exchange_rates
		WHERE from_currency = $1 AND to_currency = 'COP' AND rate_date <= $2
		ORDER BY rate_date DESC
		LIMIT 1`,
		fromCurrency, date.Format("2006-01-02"),
	).Scan(&er.ID, &er.Date, &er.FromCurrency, &er.ToCurrency, &er.RateX10000, &er.Source, &er.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &er, nil
}

func (r *PostgresRepository) ListRates(ctx context.Context, from, to time.Time, fromCurrency string) ([]*ExchangeRate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, rate_date, from_currency, to_currency, rate_x10000, source, created_at
		FROM accounting.exchange_rates
		WHERE from_currency = $1 AND to_currency = 'COP'
		  AND rate_date BETWEEN $2 AND $3
		ORDER BY rate_date DESC`,
		fromCurrency, from.Format("2006-01-02"), to.Format("2006-01-02"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ExchangeRate
	for rows.Next() {
		var er ExchangeRate
		if err := rows.Scan(&er.ID, &er.Date, &er.FromCurrency, &er.ToCurrency, &er.RateX10000, &er.Source, &er.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &er)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) RevaluationBalances(ctx context.Context, companyID uuid.UUID, currency string) ([]revalBalance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
		    jl.account_id,
		    a.code,
		    a.name,
		    SUM(CASE WHEN jl.debit > 0 THEN jl.foreign_amount ELSE -jl.foreign_amount END) AS foreign_balance,
		    SUM(jl.debit - jl.credit)                                                       AS cop_balance
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		JOIN accounting.accounts        a  ON a.id  = jl.account_id
		WHERE jl.foreign_currency = $1
		  AND je.company_id       = $2
		  AND je.status           = 'POSTED'
		  AND jl.foreign_amount  IS NOT NULL
		  AND jl.foreign_amount   != 0
		GROUP BY jl.account_id, a.code, a.name
		HAVING SUM(CASE WHEN jl.debit > 0 THEN jl.foreign_amount ELSE -jl.foreign_amount END) != 0`,
		currency, companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []revalBalance
	for rows.Next() {
		var b revalBalance
		if err := rows.Scan(&b.AccountID, &b.AccountCode, &b.AccountName, &b.Foreign, &b.COP); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
