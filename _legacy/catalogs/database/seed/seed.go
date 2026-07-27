package seed

import (
	"context"
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.csv
var seedFS embed.FS

// All carga los 13 catálogos DIAN/DANE desde los CSVs embebidos.
// Es idempotente: ON CONFLICT DO UPDATE actualiza sin duplicar.
func All(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []struct {
		name string
		csv  string
		cols []string
	}{
		{"catalogs.currencies", "currencies", []string{"code", "name", "symbol"}},
		{"catalogs.departments", "departments", []string{"code", "name", "description"}},
		{"catalogs.identification_types", "identification_types", []string{"code", "name", "description"}},
		{"catalogs.municipalities", "municipalities", []string{"code", "name", "department_code", "description"}},
		{"catalogs.payment_methods", "payment_methods", []string{"code", "name", "description"}},
		{"catalogs.dian_tax_types", "dian_tax_types", []string{"code", "name", "description"}},
		{"catalogs.unit_measures", "unit_measures", []string{"code", "name", "description"}},
		{"catalogs.dian_document_types", "dian_document_types", []string{"code", "name", "description"}},
		{"catalogs.payment_terms", "payment_terms", []string{"code", "name", "description"}},
		{"catalogs.tax_regimes", "tax_regimes", []string{"code", "name", "description"}},
		{"catalogs.liability_codes", "liability_codes", []string{"code", "name", "description"}},
		{"catalogs.item_standards", "item_standards", []string{"code", "name", "agency_id", "description"}},
		{"catalogs.ciiu_codes", "ciiu_codes", []string{"code", "description"}},
	}

	for _, t := range tables {
		if err := seedTable(ctx, pool, t.name, t.csv, t.cols); err != nil {
			return fmt.Errorf("seed %s: %w", t.name, err)
		}
	}
	return nil
}

func seedTable(ctx context.Context, pool *pgxpool.Pool, table, csvName string, cols []string) error {
	rows, err := readCSV(csvName, cols)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	query := buildUpsert(table, cols)
	batch := &pgx.Batch{}
	for _, row := range rows {
		args := make([]any, len(cols))
		for i, c := range cols {
			args[i] = row[c]
		}
		batch.Queue(query, args...)
	}

	br := pool.SendBatch(ctx, batch)
	defer br.Close()
	for range rows {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return br.Close()
}

func readCSV(name string, cols []string) ([]map[string]string, error) {
	f, err := seedFS.Open(name + ".csv")
	if err != nil {
		return nil, fmt.Errorf("open %s.csv: %w", name, err)
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
		row := make(map[string]string, len(cols))
		for _, c := range cols {
			row[c] = rec[idx[c]]
		}
		out = append(out, row)
	}
	return out, nil
}

func buildUpsert(table string, cols []string) string {
	placeholders := ""
	updates := ""
	for i, c := range cols {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf("$%d", i+1)
		if c == "code" {
			continue
		}
		if updates != "" {
			updates += ", "
		}
		updates += fmt.Sprintf("%s = EXCLUDED.%s", c, c)
	}
	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (code) DO UPDATE SET %s",
		table, strings.Join(cols, ", "), placeholders, updates,
	)
}
