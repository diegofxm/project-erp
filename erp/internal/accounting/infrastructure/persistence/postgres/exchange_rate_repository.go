package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type ExchangeRateRepository struct {
	pool *pgxpool.Pool
}

func NewExchangeRateRepository(pool *pgxpool.Pool) *ExchangeRateRepository {
	return &ExchangeRateRepository{pool: pool}
}

func (r *ExchangeRateRepository) Set(ctx context.Context, rate domain.ExchangeRate) (*domain.ExchangeRate, error) {
	const q = `
		INSERT INTO accounting.exchange_rates (rate_date, from_currency, to_currency, rate_x10000, source, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (rate_date, from_currency, to_currency) DO UPDATE
		SET rate_x10000 = EXCLUDED.rate_x10000, source = EXCLUDED.source, description = EXCLUDED.description
		RETURNING rate_date, from_currency, to_currency, rate_x10000, source, description, created_at`
	var out domain.ExchangeRate
	err := r.pool.QueryRow(ctx, q, rate.RateDate, rate.FromCurrency, rate.ToCurrency, rate.RateX10000, rate.Source, rate.Description).
		Scan(&out.RateDate, &out.FromCurrency, &out.ToCurrency, &out.RateX10000, &out.Source, &out.Description, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("guardar tasa de cambio: %w", err)
	}
	return &out, nil
}

func (r *ExchangeRateRepository) Get(ctx context.Context, date time.Time, from, to string) (*domain.ExchangeRate, error) {
	const q = `
		SELECT rate_date, from_currency, to_currency, rate_x10000, source, description, created_at
		FROM accounting.exchange_rates
		WHERE rate_date = $1 AND from_currency = $2 AND to_currency = $3`
	var out domain.ExchangeRate
	err := r.pool.QueryRow(ctx, q, date, from, to).
		Scan(&out.RateDate, &out.FromCurrency, &out.ToCurrency, &out.RateX10000, &out.Source, &out.Description, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrExchangeRateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("leer tasa de cambio: %w", err)
	}
	return &out, nil
}

func (r *ExchangeRateRepository) List(ctx context.Context, from, to time.Time) ([]domain.ExchangeRate, error) {
	const q = `
		SELECT rate_date, from_currency, to_currency, rate_x10000, source, description, created_at
		FROM accounting.exchange_rates
		WHERE rate_date BETWEEN $1 AND $2
		ORDER BY rate_date DESC, from_currency`
	rows, err := r.pool.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("listar tasas de cambio: %w", err)
	}
	defer rows.Close()
	var out []domain.ExchangeRate
	for rows.Next() {
		var e domain.ExchangeRate
		if err := rows.Scan(&e.RateDate, &e.FromCurrency, &e.ToCurrency, &e.RateX10000, &e.Source, &e.Description, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
