package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/payroll/domain"
)

type EmployeeRepository struct {
	pool *pgxpool.Pool
}

func NewEmployeeRepository(pool *pgxpool.Pool) *EmployeeRepository {
	return &EmployeeRepository{pool: pool}
}

const employeeCols = `id, company_id, identification_type_code, identification_number,
	first_name, last_name, email, phone,
	department_code, municipality_code, address_line,
	is_active, created_at, updated_at`

func (r *EmployeeRepository) Create(ctx context.Context, in domain.CreateEmployeeInput) (*domain.Employee, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO payroll.employees
		 (company_id, identification_type_code, identification_number,
		  first_name, last_name, email, phone,
		  department_code, municipality_code, address_line)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING `+employeeCols,
		in.CompanyID, in.IdentificationTypeCode, in.IdentificationNumber,
		in.FirstName, in.LastName, nilStr(in.Email), nilStr(in.Phone),
		nilStr(in.DepartmentCode), nilStr(in.MunicipalityCode), nilStr(in.AddressLine),
	)
	e, err := scanEmployee(row)
	if err != nil {
		if isUniqueViol(err) {
			return nil, domain.ErrEmployeeExists
		}
		return nil, fmt.Errorf("employees create: %w", err)
	}
	return e, nil
}

func (r *EmployeeRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+employeeCols+` FROM payroll.employees WHERE id = $1`, id)
	e, err := scanEmployee(row)
	if err != nil {
		return nil, fmt.Errorf("employees get: %w", err)
	}
	return e, nil
}

func (r *EmployeeRepository) ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*domain.Employee, error) {
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

	var out []*domain.Employee
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *EmployeeRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE payroll.employees SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("employees deactivate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEmployeeNotFound
	}
	return nil
}

func scanEmployee(row pgx.Row) (*domain.Employee, error) {
	var e domain.Employee
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
		return nil, domain.ErrEmployeeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan employee: %w", err)
	}
	derefStr(&e.Email, email)
	derefStr(&e.Phone, phone)
	derefStr(&e.DepartmentCode, deptCode)
	derefStr(&e.MunicipalityCode, munCode)
	derefStr(&e.AddressLine, addrLine)
	return &e, nil
}

func isUniqueViol(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nilStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefStr(dst *string, src *string) {
	if src != nil {
		*dst = *src
	}
}
