package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/hr/domain"
)

type AbsenceRepository struct {
	pool *pgxpool.Pool
}

func NewAbsenceRepository(pool *pgxpool.Pool) *AbsenceRepository {
	return &AbsenceRepository{pool: pool}
}

const absenceCols = `id, company_id, employee_id, type, status,
	start_date, end_date, days, reason, notes, created_at, updated_at`

func (r *AbsenceRepository) Create(ctx context.Context, in domain.CreateAbsenceInput) (*domain.Absence, error) {
	days := int(in.EndDate.Sub(in.StartDate).Hours()/24) + 1
	row := r.pool.QueryRow(ctx,
		`INSERT INTO hr.absences
		 (company_id, employee_id, type, start_date, end_date, days, reason)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING `+absenceCols,
		in.CompanyID, in.EmployeeID, in.Type,
		in.StartDate, in.EndDate, days, nilStr(in.Reason),
	)
	return scanAbsence(row)
}

func (r *AbsenceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Absence, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+absenceCols+` FROM hr.absences WHERE id = $1`, id)
	a, err := scanAbsence(row)
	if err != nil {
		return nil, fmt.Errorf("hr absences get: %w", err)
	}
	return a, nil
}

func (r *AbsenceRepository) ListByEmployee(ctx context.Context, companyID, employeeID uuid.UUID) ([]*domain.Absence, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+absenceCols+`
		 FROM hr.absences
		 WHERE company_id = $1 AND employee_id = $2
		 ORDER BY start_date DESC`,
		companyID, employeeID,
	)
	if err != nil {
		return nil, fmt.Errorf("hr absences list by employee: %w", err)
	}
	defer rows.Close()
	return collectAbsences(rows)
}

func (r *AbsenceRepository) ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*domain.Absence, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+absenceCols+`
		 FROM hr.absences
		 WHERE company_id = $1
		 ORDER BY start_date DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("hr absences list by company: %w", err)
	}
	defer rows.Close()
	return collectAbsences(rows)
}

func (r *AbsenceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AbsenceStatus, notes string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE hr.absences SET status = $2, notes = $3, updated_at = NOW() WHERE id = $1`,
		id, status, nilStr(notes),
	)
	if err != nil {
		return fmt.Errorf("hr absences update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAbsenceNotFound
	}
	return nil
}

func scanAbsence(row pgx.Row) (*domain.Absence, error) {
	var a domain.Absence
	var reason, notes *string
	err := row.Scan(
		&a.ID, &a.CompanyID, &a.EmployeeID, &a.Type, &a.Status,
		&a.StartDate, &a.EndDate, &a.Days,
		&reason, &notes,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAbsenceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan absence: %w", err)
	}
	if reason != nil {
		a.Reason = *reason
	}
	if notes != nil {
		a.Notes = *notes
	}
	return &a, nil
}

func collectAbsences(rows pgx.Rows) ([]*domain.Absence, error) {
	var out []*domain.Absence
	for rows.Next() {
		a, err := scanAbsence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func nilStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
