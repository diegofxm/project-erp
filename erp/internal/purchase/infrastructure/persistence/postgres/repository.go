package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/purchase/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Save(ctx context.Context, o domain.PurchaseOrder) (*domain.PurchaseOrder, error) {
	now := time.Now()
	o.CreatedAt = now
	o.UpdatedAt = now

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO purchase.orders (id, company_id, supplier_id, number, status, issue_date, due_date, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		o.ID, o.CompanyID, o.SupplierID, o.Number, string(o.Status),
		o.IssueDate, o.DueDate, o.Notes, o.CreatedAt, o.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar orden: %w", err)
	}

	for i := range o.Lines {
		l := &o.Lines[i]
		l.ID = uuid.New()
		l.PurchaseOrderID = o.ID
		_, err = tx.Exec(ctx, `
			INSERT INTO purchase.order_lines (id, purchase_order_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			l.ID, l.PurchaseOrderID, l.ProductID, l.Description,
			l.Quantity, l.UnitPrice, l.TaxRate, l.Subtotal, l.TaxAmount, l.Total,
		)
		if err != nil {
			return nil, fmt.Errorf("guardar línea: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &o, nil
}

func (r *Repository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.PurchaseOrder, error) {
	var o domain.PurchaseOrder
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT id, company_id, supplier_id, number, status, issue_date, due_date, notes, created_at, updated_at
		FROM purchase.orders WHERE id=$1 AND company_id=$2`,
		id, companyID,
	).Scan(&o.ID, &o.CompanyID, &o.SupplierID, &o.Number, &status,
		&o.IssueDate, &o.DueDate, &o.Notes, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPurchaseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener orden: %w", err)
	}
	o.Status = domain.PurchaseStatus(status)

	lines, err := r.loadLines(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Lines = lines
	return &o, nil
}

func (r *Repository) List(ctx context.Context, companyID uuid.UUID) ([]domain.PurchaseOrder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, supplier_id, number, status, issue_date, due_date, notes, created_at, updated_at
		FROM purchase.orders WHERE company_id=$1 ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar órdenes: %w", err)
	}
	defer rows.Close()

	var out []domain.PurchaseOrder
	for rows.Next() {
		var o domain.PurchaseOrder
		var status string
		if err := rows.Scan(&o.ID, &o.CompanyID, &o.SupplierID, &o.Number, &status,
			&o.IssueDate, &o.DueDate, &o.Notes, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("leer orden: %w", err)
		}
		o.Status = domain.PurchaseStatus(status)
		out = append(out, o)
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

func (r *Repository) UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status domain.PurchaseStatus) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE purchase.orders SET status=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3",
		string(status), id, companyID,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	o, err := r.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if o.Status != domain.StatusDraft {
		return domain.ErrPurchaseNotDraft
	}
	_, err = r.pool.Exec(ctx, "DELETE FROM purchase.orders WHERE id=$1 AND company_id=$2", id, companyID)
	return err
}

func (r *Repository) loadLines(ctx context.Context, orderID uuid.UUID) ([]domain.PurchaseLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, purchase_order_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total
		FROM purchase.order_lines WHERE purchase_order_id=$1`,
		orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("cargar líneas: %w", err)
	}
	defer rows.Close()

	var lines []domain.PurchaseLine
	for rows.Next() {
		var l domain.PurchaseLine
		if err := rows.Scan(&l.ID, &l.PurchaseOrderID, &l.ProductID, &l.Description,
			&l.Quantity, &l.UnitPrice, &l.TaxRate, &l.Subtotal, &l.TaxAmount, &l.Total); err != nil {
			return nil, fmt.Errorf("leer línea: %w", err)
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}
