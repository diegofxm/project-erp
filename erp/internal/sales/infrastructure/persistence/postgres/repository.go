package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/sales/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Save(ctx context.Context, s domain.Sale) (*domain.Sale, error) {
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO sales.sales (id, company_id, customer_id, number, status, issue_date, due_date, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		s.ID, s.CompanyID, s.CustomerID, s.Number, string(s.Status),
		s.IssueDate, s.DueDate, s.Notes, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar venta: %w", err)
	}

	for i := range s.Lines {
		l := &s.Lines[i]
		l.ID = uuid.New()
		l.SaleID = s.ID
		_, err = tx.Exec(ctx, `
			INSERT INTO sales.sale_lines (id, sale_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			l.ID, l.SaleID, l.ProductID, l.Description,
			l.Quantity, l.UnitPrice, l.TaxRate, l.Subtotal, l.TaxAmount, l.Total, l.Discount,
		)
		if err != nil {
			return nil, fmt.Errorf("guardar línea: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &s, nil
}

func (r *Repository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Sale, error) {
	var s domain.Sale
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT id, company_id, customer_id, number, status, issue_date, due_date, notes, invoice_document_id, created_at, updated_at
		FROM sales.sales WHERE id=$1 AND company_id=$2`,
		id, companyID,
	).Scan(&s.ID, &s.CompanyID, &s.CustomerID, &s.Number, &status,
		&s.IssueDate, &s.DueDate, &s.Notes, &s.InvoiceDocumentID, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSaleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener venta: %w", err)
	}
	s.Status = domain.SaleStatus(status)

	lines, err := r.loadLines(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.Lines = lines
	return &s, nil
}

func (r *Repository) List(ctx context.Context, companyID uuid.UUID) ([]domain.Sale, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, customer_id, number, status, issue_date, due_date, notes, invoice_document_id, created_at, updated_at
		FROM sales.sales WHERE company_id=$1 ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar ventas: %w", err)
	}
	defer rows.Close()

	var out []domain.Sale
	for rows.Next() {
		var s domain.Sale
		var status string
		if err := rows.Scan(&s.ID, &s.CompanyID, &s.CustomerID, &s.Number, &status,
			&s.IssueDate, &s.DueDate, &s.Notes, &s.InvoiceDocumentID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("leer venta: %w", err)
		}
		s.Status = domain.SaleStatus(status)
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		lines, err := r.loadLines(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Lines = lines
	}
	return out, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status domain.SaleStatus) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE sales.sales SET status=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3",
		string(status), id, companyID,
	)
	return err
}

func (r *Repository) SetInvoiceDocumentID(ctx context.Context, companyID, id, documentID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE sales.sales SET invoice_document_id=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3",
		documentID, id, companyID,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	s, err := r.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if s.Status != domain.StatusDraft {
		return domain.ErrSaleNotDraft
	}
	_, err = r.pool.Exec(ctx, "DELETE FROM sales.sales WHERE id=$1 AND company_id=$2", id, companyID)
	return err
}

func (r *Repository) loadLines(ctx context.Context, saleID uuid.UUID) ([]domain.SaleLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sale_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount
		FROM sales.sale_lines WHERE sale_id=$1`,
		saleID,
	)
	if err != nil {
		return nil, fmt.Errorf("cargar líneas: %w", err)
	}
	defer rows.Close()

	var lines []domain.SaleLine
	for rows.Next() {
		var l domain.SaleLine
		if err := rows.Scan(&l.ID, &l.SaleID, &l.ProductID, &l.Description,
			&l.Quantity, &l.UnitPrice, &l.TaxRate, &l.Subtotal, &l.TaxAmount, &l.Total, &l.Discount); err != nil {
			return nil, fmt.Errorf("leer línea: %w", err)
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

// NextSaleNumber asigna el siguiente consecutivo de venta para la empresa y el año dados —
// arranca en 1 cada año, comparte la tabla con NextQuoteNumber (doc_type='sale').
func (r *Repository) NextSaleNumber(ctx context.Context, companyID uuid.UUID, year int) (int, error) {
	const q = `
		INSERT INTO sales.number_counters (company_id, doc_type, year, last_seq)
		VALUES ($1, 'sale', $2, 1)
		ON CONFLICT (company_id, doc_type, year)
		DO UPDATE SET last_seq = sales.number_counters.last_seq + 1
		RETURNING last_seq`
	var seq int
	if err := r.pool.QueryRow(ctx, q, companyID, year).Scan(&seq); err != nil {
		return 0, fmt.Errorf("asignar consecutivo de venta: %w", err)
	}
	return seq, nil
}
