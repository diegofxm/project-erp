package database

import (
	"context"
	"embed"
	"encoding/csv"
	"fmt"
	"io"

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
		name string
		cols []string
	}{
		{"currencies", []string{"code", "name", "symbol"}},
		{"departments", []string{"code", "name", "description"}},
		{"identification_types", []string{"code", "name", "description"}},
		{"municipalities", []string{"code", "name", "department_code", "description"}},
		{"payment_methods", []string{"code", "name", "description"}},
		{"tax_types", []string{"code", "name", "description"}},
		{"unit_measures", []string{"code", "name", "description"}},
		{"dian_document_types", []string{"code", "name", "description"}},
		{"payment_terms", []string{"code", "name", "description"}},
		{"tax_regimes", []string{"code", "name", "description"}},
		{"liability_codes", []string{"code", "name", "description"}},
		{"item_standards", []string{"code", "name", "agency_id", "description"}},
	}

	for _, t := range tables {
		if err := d.seedTable(ctx, t.name, t.cols); err != nil {
			return fmt.Errorf("seed %s: %w", t.name, err)
		}
	}

	return nil
}

func (d *DB) seedTable(ctx context.Context, table string, cols []string) error {
	rows, err := readSeedCSV(table, cols)
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
