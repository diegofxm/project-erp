package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/purchase/domain"
)

type PaymentRepository struct{ pool *pgxpool.Pool }

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

var _ domain.PaymentRepository = (*PaymentRepository)(nil)

func (r *PaymentRepository) Save(ctx context.Context, p domain.PurchasePayment) (*domain.PurchasePayment, error) {
	p.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO purchase.purchase_payments (id, company_id, purchase_id, payment_date, amount, payment_method, reference, notes, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ID, p.CompanyID, p.PurchaseID, p.PaymentDate, p.Amount,
		string(p.PaymentMethod), p.Reference, p.Notes, p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar pago: %w", err)
	}
	return &p, nil
}

func (r *PaymentRepository) ListByPurchase(ctx context.Context, companyID, purchaseID uuid.UUID) ([]domain.PurchasePayment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, purchase_id, payment_date, amount, payment_method, reference, notes, created_at
		FROM purchase.purchase_payments WHERE company_id=$1 AND purchase_id=$2 ORDER BY payment_date`,
		companyID, purchaseID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar pagos: %w", err)
	}
	defer rows.Close()

	var out []domain.PurchasePayment
	for rows.Next() {
		var p domain.PurchasePayment
		var method string
		if err := rows.Scan(&p.ID, &p.CompanyID, &p.PurchaseID, &p.PaymentDate,
			&p.Amount, &method, &p.Reference, &p.Notes, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("leer pago: %w", err)
		}
		p.PaymentMethod = domain.PaymentMethod(method)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PaymentRepository) GetPayables(ctx context.Context, companyID uuid.UUID) ([]domain.PayableBalance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			o.id,
			o.number,
			o.supplier_id,
			o.issue_date,
			o.due_date,
			COALESCE(SUM(ol.total), 0) AS total,
			COALESCE(SUM(p.amount), 0) AS paid
		FROM purchase.orders o
		LEFT JOIN purchase.order_lines ol ON ol.purchase_order_id = o.id
		LEFT JOIN purchase.purchase_payments p ON p.purchase_id = o.id AND p.company_id = o.company_id
		WHERE o.company_id = $1 AND o.status = 'received'
		GROUP BY o.id, o.number, o.supplier_id, o.issue_date, o.due_date
		HAVING COALESCE(SUM(ol.total), 0) > COALESCE(SUM(p.amount), 0)
		ORDER BY o.due_date NULLS LAST, o.issue_date`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("obtener cuentas por pagar: %w", err)
	}
	defer rows.Close()

	var out []domain.PayableBalance
	for rows.Next() {
		var b domain.PayableBalance
		if err := rows.Scan(&b.PurchaseID, &b.PurchaseNumber, &b.SupplierID,
			&b.IssueDate, &b.DueDate, &b.Total, &b.Paid); err != nil {
			return nil, fmt.Errorf("leer cuenta por pagar: %w", err)
		}
		b.Balance = b.Total - b.Paid
		out = append(out, b)
	}
	return out, rows.Err()
}
