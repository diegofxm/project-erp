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

// Accounts carga el PUC completo en accounting.accounts. Es idempotente:
// ON CONFLICT (code) DO UPDATE reemplaza el nombre y flags sin duplicar filas.
func Accounts(ctx context.Context, pool *pgxpool.Pool) error {
	f, err := seedFS.Open("accounts.csv")
	if err != nil {
		return fmt.Errorf("seed accounts: open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // tolera trailing comma del CSV fuente
	if _, err := r.Read(); err != nil {
		return fmt.Errorf("seed accounts: read header: %w", err)
	}

	type row struct {
		code       string
		name       string
		parentCode *string
		level      int
		category   string
		isPosting  bool
		isActive   bool
	}

	var rows []row
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

		rows = append(rows, row{
			code:       rec[0],
			name:       rec[1],
			parentCode: parentCode,
			level:      level,
			category:   rec[4],
			isPosting:  isPosting,
			isActive:   isActive,
		})
	}

	for _, rw := range rows {
		_, err := pool.Exec(ctx, `
			INSERT INTO accounting.accounts (code, name, parent_code, level, category, is_posting, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (code) DO UPDATE SET
				name       = EXCLUDED.name,
				parent_code = EXCLUDED.parent_code,
				level      = EXCLUDED.level,
				category   = EXCLUDED.category,
				is_posting = EXCLUDED.is_posting,
				is_active  = EXCLUDED.is_active,
				updated_at = NOW()`,
			rw.code, rw.name, rw.parentCode, rw.level, rw.category, rw.isPosting, rw.isActive,
		)
		if err != nil {
			return fmt.Errorf("seed accounts: upsert %q: %w", rw.code, err)
		}
	}

	return nil
}
