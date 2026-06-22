package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diegofxm/api-dian/internal/sqlutil"
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

const userColumns = `id, issuer_id, email, password_hash, name, role, is_active, created_at, updated_at`

const userSelect = `SELECT ` + userColumns + ` FROM users`

func (r *PostgresRepository) Create(ctx context.Context, u User) (*User, error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	args := []any{u.ID, u.IssuerID, u.Email, u.PasswordHash, u.Name, u.Role, u.IsActive, u.CreatedAt, u.UpdatedAt}
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

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.IssuerID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}

func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
