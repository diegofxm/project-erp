package prospects

import (
	"context"
	"errors"

	"github.com/diegofxm/apidian/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const prospectCols = `
	id, name, email, nit,
	(cedula_pdf IS NOT NULL) AS has_cedula,
	(rut_pdf IS NOT NULL) AS has_rut,
	status, notes, reviewed_at,
	created_at, updated_at`

func scanProspect(row pgx.Row) (*Prospect, error) {
	var p Prospect
	var nit, notes *string
	err := row.Scan(
		&p.ID, &p.Name, &p.Email, &nit,
		&p.HasCedula, &p.HasRut,
		&p.Status, &notes, &p.ReviewedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if nit != nil {
		p.NIT = *nit
	}
	if notes != nil {
		p.Notes = *notes
	}
	return &p, nil
}

func isDuplicateEmail(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *PostgresRepository) Create(ctx context.Context, p Prospect, cedulaPDF, rutPDF []byte) (*Prospect, error) {
	var nit *string
	if p.NIT != "" {
		nit = &p.NIT
	}
	var cedula, rut interface{}
	if len(cedulaPDF) > 0 {
		cedula = cedulaPDF
	}
	if len(rutPDF) > 0 {
		rut = rutPDF
	}
	row := r.db.Pool.QueryRow(ctx, `
		INSERT INTO prospects (name, email, nit, cedula_pdf, rut_pdf)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+prospectCols,
		p.Name, p.Email, nit, cedula, rut,
	)
	created, err := scanProspect(row)
	if err != nil {
		if isDuplicateEmail(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}
	return created, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Prospect, error) {
	row := r.db.Pool.QueryRow(ctx,
		`SELECT `+prospectCols+` FROM prospects WHERE id = $1`, id)
	return scanProspect(row)
}

func (r *PostgresRepository) List(ctx context.Context) ([]Prospect, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT `+prospectCols+` FROM prospects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Prospect
	for rows.Next() {
		p, err := scanProspect(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Approve(ctx context.Context, id uuid.UUID) (*Prospect, error) {
	row := r.db.Pool.QueryRow(ctx, `
		UPDATE prospects
		SET status = 'approved', reviewed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
		RETURNING `+prospectCols, id)
	p, err := scanProspect(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrAlreadyReviewed
		}
		return nil, err
	}
	return p, nil
}

func (r *PostgresRepository) Reject(ctx context.Context, id uuid.UUID, notes string) (*Prospect, error) {
	var n *string
	if notes != "" {
		n = &notes
	}
	row := r.db.Pool.QueryRow(ctx, `
		UPDATE prospects
		SET status = 'rejected', notes = COALESCE($2, notes), reviewed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
		RETURNING `+prospectCols, id, n)
	p, err := scanProspect(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrAlreadyReviewed
		}
		return nil, err
	}
	return p, nil
}

func (r *PostgresRepository) GetCedulaPDF(ctx context.Context, id uuid.UUID) ([]byte, error) {
	var data []byte
	err := r.db.Pool.QueryRow(ctx,
		`SELECT cedula_pdf FROM prospects WHERE id = $1`, id).Scan(&data)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (r *PostgresRepository) GetRutPDF(ctx context.Context, id uuid.UUID) ([]byte, error) {
	var data []byte
	err := r.db.Pool.QueryRow(ctx,
		`SELECT rut_pdf FROM prospects WHERE id = $1`, id).Scan(&data)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}
