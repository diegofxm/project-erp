package seed

import (
	"context"
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed smmlv.csv arl_rates.csv
var seedFS embed.FS

func All(ctx context.Context, pool *pgxpool.Pool) error {
	if err := smmlv(ctx, pool); err != nil {
		return err
	}
	return arlRates(ctx, pool)
}

func smmlv(ctx context.Context, pool *pgxpool.Pool) error {
	f, err := seedFS.Open("smmlv.csv")
	if err != nil {
		return fmt.Errorf("seed smmlv: open: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	if _, err := r.Read(); err != nil {
		return fmt.Errorf("seed smmlv: header: %w", err)
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("seed smmlv: read: %w", err)
		}
		year, _ := strconv.Atoi(rec[0])
		cents, _ := strconv.ParseInt(rec[1], 10, 64)
		if _, err := pool.Exec(ctx,
			`INSERT INTO payroll.smmlv_values (year, amount_cents)
			 VALUES ($1, $2)
			 ON CONFLICT (year) DO UPDATE SET amount_cents = EXCLUDED.amount_cents`,
			year, cents,
		); err != nil {
			return fmt.Errorf("seed smmlv: upsert año %d: %w", year, err)
		}
	}
	return nil
}

func arlRates(ctx context.Context, pool *pgxpool.Pool) error {
	f, err := seedFS.Open("arl_rates.csv")
	if err != nil {
		return fmt.Errorf("seed arl_rates: open: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	if _, err := r.Read(); err != nil {
		return fmt.Errorf("seed arl_rates: header: %w", err)
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("seed arl_rates: read: %w", err)
		}
		year, _ := strconv.Atoi(rec[0])
		riskClass := rec[1]
		rateBP, _ := strconv.Atoi(rec[2])
		if _, err := pool.Exec(ctx,
			`INSERT INTO payroll.arl_rates (year, risk_class, rate_bp)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (year, risk_class) DO UPDATE SET rate_bp = EXCLUDED.rate_bp`,
			year, riskClass, rateBP,
		); err != nil {
			return fmt.Errorf("seed arl_rates: upsert %d/%s: %w", year, riskClass, err)
		}
	}
	return nil
}
