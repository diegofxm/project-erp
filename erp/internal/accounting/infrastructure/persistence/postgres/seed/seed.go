package seed

import (
	"context"
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.csv
var seedFS embed.FS

// All carga los datos de referencia del módulo contable.
// Es idempotente: ON CONFLICT DO NOTHING o DO UPDATE según la tabla.
// Orden importa: accounts primero (voucher_types y budget_lines dependen de ella).
func All(ctx context.Context, pool *pgxpool.Pool) error {
	if err := seedAccounts(ctx, pool); err != nil {
		return fmt.Errorf("seed accounts: %w", err)
	}
	if err := seedUVT(ctx, pool); err != nil {
		return fmt.Errorf("seed uvt: %w", err)
	}
	if err := seedWithholdingConcepts(ctx, pool); err != nil {
		return fmt.Errorf("seed withholding_concepts: %w", err)
	}
	if err := seedIncomeTaxRates(ctx, pool); err != nil {
		return fmt.Errorf("seed income_tax_rates: %w", err)
	}
	return nil
}

func seedAccounts(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := readCSV("accounts.csv")
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO accounting.accounts (code, name, parent_code, level, category, is_posting, is_active)
		VALUES ($1, $2, NULLIF($3,''), $4, $5, $6, $7)
		ON CONFLICT (code) DO UPDATE
		SET name = EXCLUDED.name, parent_code = EXCLUDED.parent_code,
		    level = EXCLUDED.level, category = EXCLUDED.category,
		    is_posting = EXCLUDED.is_posting, is_active = EXCLUDED.is_active,
		    updated_at = NOW()`
	batch := &pgx.Batch{}
	for _, r := range rows {
		level, _ := strconv.Atoi(r["level"])
		isPosting := r["is_posting"] == "true"
		isActive := r["is_active"] == "true"
		batch.Queue(q, r["code"], r["name"], r["parent_code"], level, r["category"], isPosting, isActive)
	}
	return sendBatch(ctx, pool, batch, len(rows))
}

func seedUVT(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := readCSV("uvt.csv")
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO accounting.uvt_values (year, value_cents)
		VALUES ($1, $2)
		ON CONFLICT (year) DO UPDATE SET value_cents = EXCLUDED.value_cents`
	batch := &pgx.Batch{}
	for _, r := range rows {
		year, _ := strconv.Atoi(r["year"])
		cents, _ := strconv.ParseInt(r["value_cents"], 10, 64)
		batch.Queue(q, year, cents)
	}
	return sendBatch(ctx, pool, batch, len(rows))
}

func seedWithholdingConcepts(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := readCSV("withholding_concepts.csv")
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO accounting.withholding_concepts
			(code, name, type, rate_bp, min_base_uvt, account_payable, account_receivable, applicable_to)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (code, type, applicable_to) DO UPDATE
		SET name = EXCLUDED.name, rate_bp = EXCLUDED.rate_bp,
		    min_base_uvt = EXCLUDED.min_base_uvt,
		    account_payable = EXCLUDED.account_payable,
		    account_receivable = EXCLUDED.account_receivable,
		    updated_at = NOW()`
	batch := &pgx.Batch{}
	for _, r := range rows {
		rateBP, _ := strconv.Atoi(r["rate_bp"])
		minBase, _ := strconv.ParseFloat(r["min_base_uvt"], 64)
		batch.Queue(q, r["code"], r["name"], r["type"], rateBP, minBase,
			r["account_payable"], r["account_receivable"], r["applicable_to"])
	}
	return sendBatch(ctx, pool, batch, len(rows))
}

func seedIncomeTaxRates(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := readCSV("income_tax_rates.csv")
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO accounting.income_tax_rates (year, rate_bp)
		VALUES ($1, $2)
		ON CONFLICT (year) DO UPDATE SET rate_bp = EXCLUDED.rate_bp`
	batch := &pgx.Batch{}
	for _, r := range rows {
		year, _ := strconv.Atoi(r["year"])
		rateBP, _ := strconv.Atoi(r["rate_bp"])
		batch.Queue(q, year, rateBP)
	}
	return sendBatch(ctx, pool, batch, len(rows))
}

func sendBatch(ctx context.Context, pool *pgxpool.Pool, batch *pgx.Batch, n int) error {
	if n == 0 {
		return nil
	}
	br := pool.SendBatch(ctx, batch)
	defer br.Close()
	for range n {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return br.Close()
}

func readCSV(name string) ([]map[string]string, error) {
	f, err := seedFS.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("header %s: %w", name, err)
	}
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[h] = i
	}

	var out []map[string]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %s: %w", name, err)
		}
		row := make(map[string]string, len(header))
		for _, h := range header {
			row[h] = rec[idx[h]]
		}
		out = append(out, row)
	}
	return out, nil
}
