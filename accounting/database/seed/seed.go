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

//go:embed accounts.csv withholding_concepts.csv uvt.csv
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

// WithholdingConcepts carga el catálogo de conceptos de retención.
// Es idempotente: ON CONFLICT (code, type, applicable_to) DO UPDATE.
func WithholdingConcepts(ctx context.Context, pool *pgxpool.Pool) error {
	f, err := seedFS.Open("withholding_concepts.csv")
	if err != nil {
		return fmt.Errorf("seed withholding concepts: open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	if _, err := r.Read(); err != nil { // skip header
		return fmt.Errorf("seed withholding concepts: read header: %w", err)
	}

	var (
		codes               []string
		names               []string
		types               []string
		ratesBP             []int32
		minBaseUVTs         []float64
		accountsPayable     []string
		accountsReceivable  []string
		applicableTos       []string
	)

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("seed withholding concepts: read row: %w", err)
		}
		if len(rec) < 8 {
			continue
		}
		rateBP, _ := strconv.Atoi(rec[3])
		minUVT, _ := strconv.ParseFloat(rec[4], 64)

		codes = append(codes, rec[0])
		names = append(names, rec[1])
		types = append(types, rec[2])
		ratesBP = append(ratesBP, int32(rateBP))
		minBaseUVTs = append(minBaseUVTs, minUVT)
		accountsPayable = append(accountsPayable, rec[5])
		accountsReceivable = append(accountsReceivable, rec[6])
		applicableTos = append(applicableTos, rec[7])
	}

	if len(codes) == 0 {
		return nil
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO accounting.withholding_concepts
			(code, name, type, rate_bp, min_base_uvt, account_payable, account_receivable, applicable_to)
		SELECT
			UNNEST($1::text[]),
			UNNEST($2::text[]),
			UNNEST($3::text[]),
			UNNEST($4::int[]),
			UNNEST($5::numeric[]),
			UNNEST($6::text[]),
			UNNEST($7::text[]),
			UNNEST($8::text[])
		ON CONFLICT (code, type, applicable_to) DO UPDATE SET
			name                = EXCLUDED.name,
			rate_bp             = EXCLUDED.rate_bp,
			min_base_uvt        = EXCLUDED.min_base_uvt,
			account_payable     = EXCLUDED.account_payable,
			account_receivable  = EXCLUDED.account_receivable,
			updated_at          = NOW()`,
		codes, names, types, ratesBP, minBaseUVTs, accountsPayable, accountsReceivable, applicableTos,
	)
	if err != nil {
		return fmt.Errorf("seed withholding concepts: bulk upsert: %w", err)
	}
	return nil
}

// UVT carga los valores del UVT por año.
// Es idempotente: ON CONFLICT (year) DO UPDATE.
func UVT(ctx context.Context, pool *pgxpool.Pool) error {
	f, err := seedFS.Open("uvt.csv")
	if err != nil {
		return fmt.Errorf("seed uvt: open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	if _, err := r.Read(); err != nil {
		return fmt.Errorf("seed uvt: read header: %w", err)
	}

	var years []int32
	var valuesCents []int64

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("seed uvt: read row: %w", err)
		}
		if len(rec) < 2 {
			continue
		}
		year, _ := strconv.Atoi(rec[0])
		cents, _ := strconv.ParseInt(rec[1], 10, 64)
		years = append(years, int32(year))
		valuesCents = append(valuesCents, cents)
	}

	if len(years) == 0 {
		return nil
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO accounting.uvt_values (year, value_cents)
		SELECT UNNEST($1::int[]), UNNEST($2::bigint[])
		ON CONFLICT (year) DO UPDATE SET value_cents = EXCLUDED.value_cents`,
		years, valuesCents,
	)
	if err != nil {
		return fmt.Errorf("seed uvt: bulk upsert: %w", err)
	}
	return nil
}
