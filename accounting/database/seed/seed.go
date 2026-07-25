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

//go:embed accounts.csv
var seedFS embed.FS

// Accounts carga el PUC completo en accounting.accounts en un único bulk upsert.
// Es idempotente: ON CONFLICT (code) DO UPDATE nunca duplica filas.
func Accounts(ctx context.Context, pool *pgxpool.Pool) error {
	f, err := seedFS.Open("accounts.csv")
	if err != nil {
		return fmt.Errorf("seed accounts: open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	if _, err := r.Read(); err != nil {
		return fmt.Errorf("seed accounts: read header: %w", err)
	}

	var (
		codes       []string
		names       []string
		parentCodes []*string
		levels      []int32
		categories  []string
		isPostings  []bool
		isActives   []bool
	)

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("seed accounts: read row: %w", err)
		}
		if len(rec) < 7 {
			continue
		}

		level, _ := strconv.Atoi(rec[3])
		isPosting, _ := strconv.ParseBool(rec[5])
		isActive, _ := strconv.ParseBool(rec[6])

		var parentCode *string
		if rec[2] != "" {
			s := rec[2]
			parentCode = &s
		}

		codes = append(codes, rec[0])
		names = append(names, rec[1])
		parentCodes = append(parentCodes, parentCode)
		levels = append(levels, int32(level))
		categories = append(categories, rec[4])
		isPostings = append(isPostings, isPosting)
		isActives = append(isActives, isActive)
	}

	if len(codes) == 0 {
		return nil
	}

	// Un solo round-trip para todas las filas.
	_, err = pool.Exec(ctx, `
		INSERT INTO accounting.accounts
			(code, name, parent_code, level, category, is_posting, is_active)
		SELECT
			UNNEST($1::text[]),
			UNNEST($2::text[]),
			UNNEST($3::text[]),
			UNNEST($4::int[]),
			UNNEST($5::text[]),
			UNNEST($6::bool[]),
			UNNEST($7::bool[])
		ON CONFLICT (code) DO UPDATE SET
			name        = EXCLUDED.name,
			parent_code = EXCLUDED.parent_code,
			level       = EXCLUDED.level,
			category    = EXCLUDED.category,
			is_posting  = EXCLUDED.is_posting,
			is_active   = EXCLUDED.is_active,
			updated_at  = NOW()`,
		codes, names, parentCodes, levels, categories, isPostings, isActives,
	)
	if err != nil {
		return fmt.Errorf("seed accounts: bulk upsert: %w", err)
	}

	return nil
}
