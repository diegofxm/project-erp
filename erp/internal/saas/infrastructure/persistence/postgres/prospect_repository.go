package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type ProspectRepository struct{ pool *pgxpool.Pool }

func NewProspectRepository(pool *pgxpool.Pool) *ProspectRepository {
	return &ProspectRepository{pool: pool}
}

const prospectCols = `id, name, email, nit, cedula_file, cedula_content_type, rut_file,
	rut_content_type, status, notes, reviewed_at, created_at, updated_at`

func scanProspect(row pgx.Row) (*domain.Prospect, error) {
	var p domain.Prospect
	var status string
	err := row.Scan(
		&p.ID, &p.Name, &p.Email, &p.NIT, &p.CedulaFile, &p.CedulaContentType, &p.RUTFile,
		&p.RUTContentType, &status, &p.Notes, &p.ReviewedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.Status = domain.ProspectStatus(status)
	return &p, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *ProspectRepository) Create(ctx context.Context, p domain.Prospect) (*domain.Prospect, error) {
	p.ID = uuid.New()
	row := r.pool.QueryRow(ctx, `
		INSERT INTO saas.prospects
			(id, name, email, nit, cedula_file, cedula_content_type, rut_file, rut_content_type, status, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING `+prospectCols,
		p.ID, p.Name, p.Email, p.NIT, p.CedulaFile, p.CedulaContentType, p.RUTFile, p.RUTContentType,
		string(domain.ProspectPending), p.Notes,
	)
	saved, err := scanProspect(row)
	if isUniqueViolation(err) {
		return nil, domain.ErrEmailTaken
	}
	if err != nil {
		return nil, fmt.Errorf("crear solicitud: %w", err)
	}
	return saved, nil
}

func (r *ProspectRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Prospect, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+prospectCols+" FROM saas.prospects WHERE id=$1", id)
	p, err := scanProspect(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProspectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener solicitud: %w", err)
	}
	return p, nil
}

func (r *ProspectRepository) List(ctx context.Context) ([]domain.Prospect, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+prospectCols+" FROM saas.prospects ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("listar solicitudes: %w", err)
	}
	defer rows.Close()

	var out []domain.Prospect
	for rows.Next() {
		p, err := scanProspect(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *ProspectRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ProspectStatus, notes string) (*domain.Prospect, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE saas.prospects SET status=$2, notes=$3, reviewed_at=NOW(), updated_at=NOW()
		WHERE id=$1
		RETURNING `+prospectCols,
		id, string(status), notes,
	)
	p, err := scanProspect(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProspectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("actualizar solicitud: %w", err)
	}
	return p, nil
}
