package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diegofxm/apidian/internal/sqlutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository usando pgx. PasswordHash ya viene cifrado en una
// sola vía (bcrypt, ver password.go) — a diferencia de issuers/numbering, aquí no hay nada que
// descifrar al leer.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository crea el repositorio de usuarios sobre PostgreSQL.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const userColumns = `id, email, password_hash, name, role, is_superadmin, is_active, invite_token, invite_token_expires_at, invite_accepted_at, created_at, updated_at`

const userSelect = `SELECT ` + userColumns + ` FROM users`

func (r *PostgresRepository) Create(ctx context.Context, u User) (*User, error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	var hash *string
	if u.PasswordHash != "" {
		hash = &u.PasswordHash
	}
	args := []any{
		u.ID, u.Email, hash, u.Name, u.Role, u.IsSuperAdmin, u.IsActive,
		u.InviteToken, u.InviteTokenExpiresAt, u.InviteAcceptedAt,
		u.CreatedAt, u.UpdatedAt,
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO users (`+userColumns+`) VALUES (`+sqlutil.Placeholders(len(args))+`)`, args...)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row := r.pool.QueryRow(ctx, userSelect+` WHERE id = $1`, id)
	return scanUser(row)
}

func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	row := r.pool.QueryRow(ctx, userSelect+` WHERE email = $1`, email)
	return scanUser(row)
}

func (r *PostgresRepository) Update(ctx context.Context, u User) (*User, error) {
	u.UpdatedAt = time.Now().UTC()
	row := r.pool.QueryRow(ctx,
		`UPDATE users SET name=$1, email=$2, updated_at=$3 WHERE id=$4 RETURNING `+userColumns,
		u.Name, u.Email, u.UpdatedAt, u.ID,
	)
	updated, err := scanUser(row)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}
	return updated, nil
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	var hash *string
	err := row.Scan(
		&u.ID, &u.Email, &hash, &u.Name, &u.Role, &u.IsSuperAdmin, &u.IsActive,
		&u.InviteToken, &u.InviteTokenExpiresAt, &u.InviteAcceptedAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	if hash != nil {
		u.PasswordHash = *hash
	}
	return &u, nil
}

func (r *PostgresRepository) GetByInviteToken(ctx context.Context, token uuid.UUID) (*User, error) {
	row := r.pool.QueryRow(ctx, userSelect+` WHERE invite_token = $1`, token)
	return scanUser(row)
}

func (r *PostgresRepository) SetPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $1, invite_token = NULL, invite_token_expires_at = NULL,
		    invite_accepted_at = NOW(), updated_at = NOW()
		WHERE id = $2`,
		passwordHash, userID,
	)
	if err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListAll(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx, userSelect+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

// LinkIssuer es idempotente (ON CONFLICT DO NOTHING sobre la PK compuesta) — vincular dos
// veces el mismo par (userID, issuerID) no debe fallar.
func (r *PostgresRepository) LinkIssuer(ctx context.Context, userID, issuerID uuid.UUID, role string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_issuers (user_id, issuer_id, role, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, issuer_id) DO NOTHING`,
		userID, issuerID, role, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("link issuer: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListIssuerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT issuer_id FROM user_issuers WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list issuer ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan issuer id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresRepository) HasAccess(ctx context.Context, userID, issuerID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_issuers WHERE user_id = $1 AND issuer_id = $2)`, userID, issuerID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check issuer access: %w", err)
	}
	return exists, nil
}

func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
