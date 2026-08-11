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
			INSERT INTO purchase.order_lines (id, purchase_order_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			l.ID, l.PurchaseOrderID, l.ProductID, l.Description,
			l.Quantity, l.UnitPrice, l.TaxRate, l.Subtotal, l.TaxAmount, l.Total, l.Discount,
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
		SELECT id, company_id, supplier_id, number, status, issue_date, due_date, notes, created_at, updated_at, support_document_id
		FROM purchase.orders WHERE id=$1 AND company_id=$2`,
		id, companyID,
	).Scan(&o.ID, &o.CompanyID, &o.SupplierID, &o.Number, &status,
		&o.IssueDate, &o.DueDate, &o.Notes, &o.CreatedAt, &o.UpdatedAt, &o.SupportDocumentID)
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

	withholdings, err := r.loadWithholdings(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Withholdings = withholdings
	return &o, nil
}

func (r *Repository) List(ctx context.Context, companyID uuid.UUID) ([]domain.PurchaseOrder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, supplier_id, number, status, issue_date, due_date, notes, created_at, updated_at, support_document_id
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
			&o.IssueDate, &o.DueDate, &o.Notes, &o.CreatedAt, &o.UpdatedAt, &o.SupportDocumentID); err != nil {
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

func (r *Repository) SetSupportDocumentID(ctx context.Context, companyID, id, documentID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE purchase.orders SET support_document_id=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3",
		documentID, id, companyID,
	)
	return err
}

func (r *Repository) Update(ctx context.Context, companyID, id uuid.UUID, o domain.PurchaseOrder) (*domain.PurchaseOrder, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM purchase.orders WHERE id=$1 AND company_id=$2 FOR UPDATE`, id, companyID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPurchaseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultar orden: %w", err)
	}
	if status != string(domain.StatusDraft) {
		return nil, domain.ErrPurchaseNotDraft
	}

	now := time.Now()
	_, err = tx.Exec(ctx,
		`UPDATE purchase.orders SET supplier_id=$3, issue_date=$4, due_date=$5, notes=$6, updated_at=$7 WHERE id=$1 AND company_id=$2`,
		id, companyID, o.SupplierID, o.IssueDate, o.DueDate, o.Notes, now,
	)
	if err != nil {
		return nil, fmt.Errorf("actualizar orden: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM purchase.order_lines WHERE purchase_order_id=$1`, id); err != nil {
		return nil, fmt.Errorf("limpiar líneas: %w", err)
	}
	for i := range o.Lines {
		l := &o.Lines[i]
		l.ID = uuid.New()
		l.PurchaseOrderID = id
		if _, err := tx.Exec(ctx, `
			INSERT INTO purchase.order_lines (id, purchase_order_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			l.ID, l.PurchaseOrderID, l.ProductID, l.Description,
			l.Quantity, l.UnitPrice, l.TaxRate, l.Subtotal, l.TaxAmount, l.Total, l.Discount,
		); err != nil {
			return nil, fmt.Errorf("guardar línea: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, companyID, id)
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

func (r *Repository) loadWithholdings(ctx context.Context, orderID uuid.UUID) ([]domain.PurchaseWithholding, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, purchase_order_id, concept_code, concept_name, base, rate_bp, amount, account_payable, created_at
		FROM purchase.purchase_withholdings WHERE purchase_order_id=$1 ORDER BY created_at`,
		orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("cargar retenciones: %w", err)
	}
	defer rows.Close()

	var out []domain.PurchaseWithholding
	for rows.Next() {
		var w domain.PurchaseWithholding
		if err := rows.Scan(&w.ID, &w.PurchaseOrderID, &w.ConceptCode, &w.ConceptName,
			&w.Base, &w.RateBP, &w.Amount, &w.AccountPayable, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("leer retención: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repository) loadLines(ctx context.Context, orderID uuid.UUID) ([]domain.PurchaseLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, purchase_order_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount
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
			&l.Quantity, &l.UnitPrice, &l.TaxRate, &l.Subtotal, &l.TaxAmount, &l.Total, &l.Discount); err != nil {
			return nil, fmt.Errorf("leer línea: %w", err)
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

// NextPurchaseNumber asigna el siguiente consecutivo de orden de compra para la empresa y el año
// dados — arranca en 1 cada año. Mismo patrón que sales.Repository.NextSaleNumber.
func (r *Repository) NextPurchaseNumber(ctx context.Context, companyID uuid.UUID, year int) (int, error) {
	const q = `
		INSERT INTO purchase.number_counters (company_id, year, last_seq)
		VALUES ($1, $2, 1)
		ON CONFLICT (company_id, year)
		DO UPDATE SET last_seq = purchase.number_counters.last_seq + 1
		RETURNING last_seq`
	var seq int
	if err := r.pool.QueryRow(ctx, q, companyID, year).Scan(&seq); err != nil {
		return 0, fmt.Errorf("asignar consecutivo de orden de compra: %w", err)
	}
	return seq, nil
}

// SetPurchaseNumberCounter — ver sales.Repository.SetSaleNumberCounter, mismo comportamiento
// para órdenes de compra.
func (r *Repository) SetPurchaseNumberCounter(ctx context.Context, companyID uuid.UUID, year, nextNumber int) (int, error) {
	const q = `
		INSERT INTO purchase.number_counters (company_id, year, last_seq)
		VALUES ($1, $2, $3)
		ON CONFLICT (company_id, year)
		DO UPDATE SET last_seq = EXCLUDED.last_seq
		WHERE EXCLUDED.last_seq > purchase.number_counters.last_seq
		RETURNING last_seq`
	var seq int
	err := r.pool.QueryRow(ctx, q, companyID, year, nextNumber-1).Scan(&seq)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrNumberCounterBackwards
	}
	if err != nil {
		return 0, fmt.Errorf("fijar consecutivo de orden de compra: %w", err)
	}
	return seq, nil
}
