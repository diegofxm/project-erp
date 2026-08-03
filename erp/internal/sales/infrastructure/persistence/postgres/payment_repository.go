package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/sales/domain"
)

type PaymentRepository struct{ pool *pgxpool.Pool }

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

var _ domain.PaymentRepository = (*PaymentRepository)(nil)

func (r *PaymentRepository) Save(ctx context.Context, p domain.SalePayment) (*domain.SalePayment, error) {
	p.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sales.sale_payments (id, company_id, sale_id, payment_date, amount, payment_method, reference, notes, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ID, p.CompanyID, p.SaleID, p.PaymentDate, p.Amount,
		string(p.PaymentMethod), p.Reference, p.Notes, p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar pago: %w", err)
	}
	return &p, nil
}

func (r *PaymentRepository) ListBySale(ctx context.Context, companyID, saleID uuid.UUID) ([]domain.SalePayment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, sale_id, payment_date, amount, payment_method, reference, notes, created_at
		FROM sales.sale_payments WHERE company_id=$1 AND sale_id=$2 ORDER BY payment_date`,
		companyID, saleID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar pagos: %w", err)
	}
	defer rows.Close()

	var out []domain.SalePayment
	for rows.Next() {
		var p domain.SalePayment
		var method string
		if err := rows.Scan(&p.ID, &p.CompanyID, &p.SaleID, &p.PaymentDate,
			&p.Amount, &method, &p.Reference, &p.Notes, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("leer pago: %w", err)
		}
		p.PaymentMethod = domain.PaymentMethod(method)
		out = append(out, p)
	}
	return out, rows.Err()
}

const receivablesBaseQuery = `
	SELECT
		s.id,
		s.number,
		s.customer_id,
		s.issue_date,
		s.due_date,
		COALESCE(SUM(sl.total), 0) AS total,
		COALESCE(SUM(p.amount), 0) AS paid
	FROM sales.sales s
	LEFT JOIN sales.sale_lines sl ON sl.sale_id = s.id
	LEFT JOIN sales.sale_payments p ON p.sale_id = s.id AND p.company_id = s.company_id
	WHERE s.company_id = $1 AND s.status = 'confirmed'`

const receivablesGroupOrder = `
	GROUP BY s.id, s.number, s.customer_id, s.issue_date, s.due_date
	HAVING COALESCE(SUM(sl.total), 0) > COALESCE(SUM(p.amount), 0)
	ORDER BY s.due_date NULLS LAST, s.issue_date`

func (r *PaymentRepository) GetReceivables(ctx context.Context, companyID uuid.UUID) ([]domain.ReceivableBalance, error) {
	rows, err := r.pool.Query(ctx, receivablesBaseQuery+receivablesGroupOrder, companyID)
	if err != nil {
		return nil, fmt.Errorf("obtener cartera: %w", err)
	}
	defer rows.Close()
	return scanReceivables(rows)
}

func (r *PaymentRepository) GetReceivablesByCustomer(ctx context.Context, companyID, customerID uuid.UUID) ([]domain.ReceivableBalance, error) {
	rows, err := r.pool.Query(ctx,
		receivablesBaseQuery+" AND s.customer_id = $2"+receivablesGroupOrder,
		companyID, customerID,
	)
	if err != nil {
		return nil, fmt.Errorf("obtener cartera del cliente: %w", err)
	}
	defer rows.Close()
	return scanReceivables(rows)
}

func scanReceivables(rows pgx.Rows) ([]domain.ReceivableBalance, error) {
	var out []domain.ReceivableBalance
	for rows.Next() {
		var b domain.ReceivableBalance
		if err := rows.Scan(&b.SaleID, &b.SaleNumber, &b.CustomerID,
			&b.IssueDate, &b.DueDate, &b.Total, &b.Paid); err != nil {
			return nil, fmt.Errorf("leer cartera: %w", err)
		}
		b.Balance = b.Total - b.Paid
		out = append(out, b)
	}
	return out, rows.Err()
}
