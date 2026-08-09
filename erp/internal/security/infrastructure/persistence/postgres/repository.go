package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/security/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Save(ctx context.Context, u domain.User) (*domain.User, error) {
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO security.users
			(id, email, password_hash, name, role, is_active, is_superadmin,
			 invite_token, invite_token_expires_at, invite_accepted_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		u.ID, u.Email, u.PasswordHash, u.Name, u.Role, u.IsActive, u.IsSuperAdmin,
		u.InviteToken, u.InviteTokenExpiresAt, u.InviteAcceptedAt, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar usuario: %w", err)
	}
	return &u, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.scanOne(ctx, "WHERE id = $1", id)
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanOne(ctx, "WHERE email = $1", email)
}

func (r *Repository) GetByInviteToken(ctx context.Context, token uuid.UUID) (*domain.User, error) {
	return r.scanOne(ctx, "WHERE invite_token = $1", token)
}

func (r *Repository) UpdateProfile(ctx context.Context, id uuid.UUID, name, email string) (*domain.User, error) {
	_, err := r.pool.Exec(ctx,
		"UPDATE security.users SET name=$1, email=$2, updated_at=NOW() WHERE id=$3",
		name, email, id,
	)
	if err != nil {
		return nil, fmt.Errorf("actualizar perfil: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) SetPassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE security.users
		 SET password_hash=$1, invite_token=NULL, invite_token_expires_at=NULL,
		     invite_accepted_at=NOW(), updated_at=NOW()
		 WHERE id=$2`,
		hash, id,
	)
	if err != nil {
		return fmt.Errorf("establecer contraseña: %w", err)
	}
	return nil
}

func (r *Repository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE security.users SET password_hash=$1, updated_at=NOW() WHERE id=$2`,
		hash, id,
	)
	if err != nil {
		return fmt.Errorf("actualizar contraseña: %w", err)
	}
	return nil
}

func (r *Repository) SetInviteToken(ctx context.Context, userID uuid.UUID, token uuid.UUID, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE security.users
		 SET invite_token=$1, invite_token_expires_at=$2, updated_at=NOW()
		 WHERE id=$3`,
		token, expiresAt, userID,
	)
	if err != nil {
		return fmt.Errorf("regenerar token de invitación: %w", err)
	}
	return nil
}

// IsSuperAdmin es una consulta puntual (sin traer el resto del usuario) — la usa
// company.CreateUseCase para decidir si la empresa recién creada debe quedar con el plan Interno
// de una vez (ver company/infrastructure/saas.Adapter).
func (r *Repository) IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	var isSuperAdmin bool
	err := r.pool.QueryRow(ctx, `SELECT is_superadmin FROM security.users WHERE id=$1`, userID).Scan(&isSuperAdmin)
	if err != nil {
		return false, fmt.Errorf("verificar superadmin: %w", err)
	}
	return isSuperAdmin, nil
}

func (r *Repository) IncrementTokenVersion(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE security.users SET token_version = token_version + 1, updated_at=NOW() WHERE id=$1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("revocar sesiones: %w", err)
	}
	return nil
}

func (r *Repository) SetSuperAdmin(ctx context.Context, userID uuid.UUID, value bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE security.users SET is_superadmin=$1, updated_at=NOW() WHERE id=$2`,
		value, userID,
	)
	if err != nil {
		return fmt.Errorf("actualizar is_superadmin: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, userSelect+" ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("listar usuarios: %w", err)
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (r *Repository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, userSelect+" WHERE id = ANY($1)", ids)
	if err != nil {
		return nil, fmt.Errorf("listar usuarios por id: %w", err)
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

// --- user_companies ---

func (r *Repository) AddCompany(ctx context.Context, userID, companyID uuid.UUID, role string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO security.user_companies (user_id, company_id, role)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, company_id) DO NOTHING`,
		userID, companyID, role,
	)
	if err != nil {
		return fmt.Errorf("vincular empresa: %w", err)
	}
	return nil
}

func (r *Repository) ListCompanyIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT company_id FROM security.user_companies WHERE user_id=$1 ORDER BY created_at",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar empresas: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) GetRole(ctx context.Context, userID, companyID uuid.UUID) (string, error) {
	var role string
	err := r.pool.QueryRow(ctx,
		"SELECT role FROM security.user_companies WHERE user_id=$1 AND company_id=$2",
		userID, companyID,
	).Scan(&role)
	if err != nil {
		return "", fmt.Errorf("obtener rol: %w", err)
	}
	return role, nil
}

func (r *Repository) HasCompany(ctx context.Context, userID, companyID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM security.user_companies WHERE user_id=$1 AND company_id=$2)",
		userID, companyID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("verificar empresa: %w", err)
	}
	return ok, nil
}

// --- helpers ---

const userSelect = `
	SELECT id, email, password_hash, name, role, is_active, is_superadmin, token_version,
	       invite_token, invite_token_expires_at, invite_accepted_at, created_at, updated_at
	FROM security.users`

func (r *Repository) scanOne(ctx context.Context, where string, arg any) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, userSelect+" "+where, arg)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(s scanner) (*domain.User, error) {
	var u domain.User
	err := s.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role,
		&u.IsActive, &u.IsSuperAdmin, &u.TokenVersion,
		&u.InviteToken, &u.InviteTokenExpiresAt, &u.InviteAcceptedAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("leer usuario: %w", err)
	}
	return &u, nil
}
