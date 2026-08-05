package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type PaymentRepository struct{ pool *pgxpool.Pool }

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

const paymentCols = "id, company_id, subscription_id, type, amount_cents, note, paid_at, created_at"

func scanPayment(row pgx.Row) (*domain.Payment, error) {
	var p domain.Payment
	var pType string
	if err := row.Scan(&p.ID, &p.CompanyID, &p.SubscriptionID, &pType, &p.AmountCents, &p.Note, &p.PaidAt, &p.CreatedAt); err != nil {
		return nil, err
	}
	p.Type = domain.PaymentType(pType)
	return &p, nil
}

func (r *PaymentRepository) Create(ctx context.Context, p domain.Payment) (*domain.Payment, error) {
	p.ID = uuid.New()
	row := r.pool.QueryRow(ctx, `
		INSERT INTO saas.payments (id, company_id, subscription_id, type, amount_cents, note, paid_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING `+paymentCols,
		p.ID, p.CompanyID, p.SubscriptionID, string(p.Type), p.AmountCents, p.Note, p.PaidAt,
	)
	saved, err := scanPayment(row)
	if err != nil {
		return nil, fmt.Errorf("registrar pago: %w", err)
	}
	return saved, nil
}

func (r *PaymentRepository) ListByCompany(ctx context.Context, companyID uuid.UUID) ([]domain.Payment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+paymentCols+` FROM saas.payments WHERE company_id=$1 ORDER BY paid_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar pagos: %w", err)
	}
	defer rows.Close()

	var out []domain.Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}
