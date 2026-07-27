package employees

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository usando pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const employeeCols = `id, company_id, identification_type_code, identification_number,
	first_name, last_name, email, phone,
	department_code, municipality_code, address_line,
	is_active, created_at, updated_at`

func (r *PostgresRepository) Create(ctx context.Context, in CreateInput) (*Employee, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO payroll.employees
		 (company_id, identification_type_code, identification_number,
		  first_name, last_name, email, phone,
		  department_code, municipality_code, address_line)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING `+employeeCols,
		in.CompanyID, in.IdentificationTypeCode, in.IdentificationNumber,
		in.FirstName, in.LastName, in.Email, in.Phone,
		in.DepartmentCode, in.MunicipalityCode, in.AddressLine,
	)
	e, err := scanEmployee(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrExists
		}
		return nil, fmt.Errorf("employees create: %w", err)
	}
	return e, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id uuid.UUID) (*Employee, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+employeeCols+` FROM payroll.employees WHERE id = $1`, id)
	e, err := scanEmployee(row)
	if err != nil {
		return nil, fmt.Errorf("employees get: %w", err)
	}
	return e, nil
}

func (r *PostgresRepository) List(ctx context.Context, companyID uuid.UUID) ([]*Employee, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+employeeCols+`
		 FROM payroll.employees
		 WHERE company_id = $1 AND is_active = true
		 ORDER BY last_name, first_name`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("employees list: %w", err)
	}
	defer rows.Close()

	var out []*Employee
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE payroll.employees SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("employees deactivate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanEmployee(row pgx.Row) (*Employee, error) {
	var e Employee
	var email, phone, deptCode, munCode, addrLine *string
	err := row.Scan(
		&e.ID, &e.CompanyID,
		&e.IdentificationTypeCode, &e.IdentificationNumber,
		&e.FirstName, &e.LastName,
		&email, &phone,
		&deptCode, &munCode, &addrLine,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan employee: %w", err)
	}
	if email != nil {
		e.Email = *email
	}
	if phone != nil {
		e.Phone = *phone
	}
	if deptCode != nil {
		e.DepartmentCode = *deptCode
	}
	if munCode != nil {
		e.MunicipalityCode = *munCode
	}
	if addrLine != nil {
		e.AddressLine = *addrLine
	}
	return &e, nil
}

// isUniqueViolation detecta el error de clave única de PostgreSQL (código 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
