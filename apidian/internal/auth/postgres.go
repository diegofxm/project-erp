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

const userColumns = `id, email, password_hash, name, role, is_superadmin, is_active, created_at, updated_at`

const userSelect = `SELECT ` + userColumns + ` FROM users`

func (r *PostgresRepository) Create(ctx context.Context, u User) (*User, error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	args := []any{u.ID, u.Email, u.PasswordHash, u.Name, u.Role, u.IsSuperAdmin, u.IsActive, u.CreatedAt, u.UpdatedAt}
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
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.IsSuperAdmin, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
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
