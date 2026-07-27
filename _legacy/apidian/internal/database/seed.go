package database

import (
	"context"
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed seed/*.csv
var seedFS embed.FS

// Seed loads the DIAN reference catalogs from the embedded CSV files in seed/.
//
// Es idempotente (ON CONFLICT ... DO UPDATE): correrlo de nuevo después de reemplazar un CSV
// actualiza las filas existentes en vez de fallar, así un catálogo se refresca sin escribir
// una migración nueva. Pensado para catálogos de cualquier tamaño — currencies tiene 3 filas,
// municipalities podría tener miles — por eso usa un batch, no inserciones una por una.
//
// 2026-06-24: completado contra la "Caja de Herramientas Factura Electrónica" oficial de la
// DIAN (docs/reference/Caja de herramientas FE_V19_(v2026)/) — departments (33/33) y
// municipalities (1.122/1.122, vía DIVIPOLA) ya están completos, igual que tax_types y
// payment_methods. payment_terms/tax_regimes/liability_codes son catálogos nuevos que esa
// misma fuente permitió agregar (antes no existían ni como tabla, ver
// migrations/000002_catalogs.up.sql).
//
// 2026-06-28: unit_measures completado igual (1.081/1.081 códigos reales UN/ECE Rec. 20, tabla
// 13.3.6 — ver docs/apidian-architecture.md sección 9.46). La muestra anterior de 11 códigos
// tenía 4 inventados/equivocados (M2/M3/MON/ANN no existen en la tabla oficial — los reales
// son MTK/MTQ/LUN/ANA), confirmados rechazados de verdad por la DIAN (regla FAV05) antes de
// corregirse.
func (d *DB) Seed(ctx context.Context) error {
	tables := []struct {
		name string // tabla SQL calificada (schema.table)
		csv  string // nombre del archivo CSV en seed/ (sin extensión); vacío = última parte de name
		cols []string
	}{
		{"catalogs.currencies", "", []string{"code", "name", "symbol"}},
		{"catalogs.departments", "", []string{"code", "name", "description"}},
		{"catalogs.identification_types", "", []string{"code", "name", "description"}},
		{"catalogs.municipalities", "", []string{"code", "name", "department_code", "description"}},
		{"catalogs.payment_methods", "", []string{"code", "name", "description"}},
		{"catalogs.dian_tax_types", "dian_tax_types", []string{"code", "name", "description"}},
		{"catalogs.unit_measures", "", []string{"code", "name", "description"}},
		{"catalogs.dian_document_types", "", []string{"code", "name", "description"}},
		{"catalogs.payment_terms", "", []string{"code", "name", "description"}},
		{"catalogs.tax_regimes", "", []string{"code", "name", "description"}},
		{"catalogs.liability_codes", "", []string{"code", "name", "description"}},
		{"catalogs.item_standards", "", []string{"code", "name", "agency_id", "description"}},
		{"catalogs.ciiu_codes", "", []string{"code", "description"}},
	}

	for _, t := range tables {
		csvName := t.csv
		if csvName == "" {
			if i := strings.LastIndex(t.name, "."); i >= 0 {
				csvName = t.name[i+1:]
			} else {
				csvName = t.name
			}
		}
		if err := d.seedTable(ctx, t.name, csvName, t.cols); err != nil {
			return fmt.Errorf("seed %s: %w", t.name, err)
		}
	}

	return nil
}

func (d *DB) seedTable(ctx context.Context, table, csvName string, cols []string) error {
	rows, err := readSeedCSV(csvName, cols)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	query := buildUpsertQuery(table, cols)

	batch := &pgx.Batch{}
	for _, row := range rows {
		args := make([]any, len(cols))
		for i, c := range cols {
			args[i] = row[c]
		}
		batch.Queue(query, args...)
	}

	br := d.Pool.SendBatch(ctx, batch)
	defer br.Close()

	for range rows {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return br.Close()
}

// readSeedCSV lee seed/<table>.csv embebido y devuelve cada fila como mapa columna→valor.
func readSeedCSV(table string, cols []string) ([]map[string]string, error) {
	f, err := seedFS.Open("seed/" + table + ".csv")
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[h] = i
	}

	var rows []map[string]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}

		row := make(map[string]string, len(cols))
		for _, c := range cols {
			row[c] = rec[idx[c]]
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// buildUpsertQuery genera "INSERT INTO table (cols...) VALUES ($1, $2, ...)
// ON CONFLICT (code) DO UPDATE SET <cols sin code> = EXCLUDED.<col>".
func buildUpsertQuery(table string, cols []string) string {
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
		table, joinCols(cols), placeholders, updates,
	)
}

// SeedPlans inserta los planes iniciales del SaaS si no existen todavía. Es idempotente:
// ON CONFLICT (name) DO UPDATE actualiza descripción/límites sin cambiar el UUID ni romper
// suscripciones existentes. Los UUIDs son generados por Postgres, no hardcodeados aquí.
func (d *DB) SeedPlans(ctx context.Context) error {
	type plan struct {
		name        string
		description string
		maxDocs     *int  // nil = ilimitado
		maxIssuers  int
		priceCOP    int
	}

	maxDocs100 := 100
	maxDocs500 := 500

	plans := []plan{
		{
			name:        "Gratis",
			description: "Plan de prueba — hasta 100 documentos por mes y 1 empresa",
			maxDocs:     &maxDocs100,
			maxIssuers:  1,
			priceCOP:    0,
		},
		{
			name:        "Básico",
			description: "Hasta 500 documentos por mes y 1 empresa",
			maxDocs:     &maxDocs500,
			maxIssuers:  1,
			priceCOP:    49900,
		},
		{
			name:        "Pro",
			description: "Documentos ilimitados y hasta 5 empresas",
			maxDocs:     nil,
			maxIssuers:  5,
			priceCOP:    149900,
		},
	}

	for _, p := range plans {
		_, err := d.Pool.Exec(ctx, `
			INSERT INTO public.plans (name, description, max_documents_per_month, max_issuers, price_cop)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (name) DO UPDATE SET
				description               = EXCLUDED.description,
				max_documents_per_month   = EXCLUDED.max_documents_per_month,
				max_issuers               = EXCLUDED.max_issuers,
				price_cop                 = EXCLUDED.price_cop,
				updated_at                = NOW()
		`, p.name, p.description, p.maxDocs, p.maxIssuers, p.priceCOP)
		if err != nil {
			return fmt.Errorf("seed plan %q: %w", p.name, err)
		}
	}
	return nil
}

func joinCols(cols []string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		out += c
	}
	return out
}
