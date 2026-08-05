package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type SubscriptionRepository struct{ pool *pgxpool.Pool }

func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

const subscriptionCols = `id, company_id, plan_id, has_own_certificate, status, contracted_price_cents,
	current_period_start, current_period_end, cert_expires_at, created_at, updated_at`

func scanSubscription(row pgx.Row) (*domain.Subscription, error) {
	var s domain.Subscription
	var status string
	err := row.Scan(
		&s.ID, &s.CompanyID, &s.PlanID, &s.HasOwnCertificate, &status, &s.ContractedPriceCents,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CertExpiresAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Status = domain.SubscriptionStatus(status)
	return &s, nil
}

func (r *SubscriptionRepository) Create(ctx context.Context, s domain.Subscription) (*domain.Subscription, error) {
	s.ID = uuid.New()
	row := r.pool.QueryRow(ctx, `
		INSERT INTO saas.subscriptions
			(id, company_id, plan_id, has_own_certificate, status, contracted_price_cents,
			 current_period_start, current_period_end, cert_expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING `+subscriptionCols,
		s.ID, s.CompanyID, s.PlanID, s.HasOwnCertificate, string(s.Status), s.ContractedPriceCents,
		s.CurrentPeriodStart, s.CurrentPeriodEnd, s.CertExpiresAt,
	)
	saved, err := scanSubscription(row)
	if err != nil {
		return nil, fmt.Errorf("crear suscripción: %w", err)
	}
	return saved, nil
}

func (r *SubscriptionRepository) GetActive(ctx context.Context, companyID uuid.UUID) (*domain.Subscription, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+subscriptionCols+`
		FROM saas.subscriptions WHERE company_id=$1 AND status='active'`,
		companyID,
	)
	s, err := scanSubscription(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener suscripción activa: %w", err)
	}
	return s, nil
}

func (r *SubscriptionRepository) Cancel(ctx context.Context, companyID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE saas.subscriptions SET status='cancelled', updated_at=NOW()
		WHERE company_id=$1 AND status='active'`, companyID,
	)
	if err != nil {
		return fmt.Errorf("cancelar suscripción: %w", err)
	}
	return nil
}

func (r *SubscriptionRepository) Renew(ctx context.Context, id uuid.UUID, newPeriodStart, newPeriodEnd time.Time, newContractedPriceCents int64) (*domain.Subscription, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE saas.subscriptions SET
			current_period_start = $2, current_period_end = $3, contracted_price_cents = $4,
			status = 'active', updated_at = NOW()
		WHERE id = $1
		RETURNING `+subscriptionCols,
		id, newPeriodStart, newPeriodEnd, newContractedPriceCents,
	)
	s, err := scanSubscription(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("renovar suscripción: %w", err)
	}
	return s, nil
}

func (r *SubscriptionRepository) ListUpcomingRenewals(ctx context.Context, withinDays int) ([]domain.Subscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+subscriptionCols+`
		FROM saas.subscriptions
		WHERE status='active' AND current_period_end <= NOW() + make_interval(days => $1)
		ORDER BY current_period_end`, withinDays,
	)
	if err != nil {
		return nil, fmt.Errorf("listar renovaciones próximas: %w", err)
	}
	defer rows.Close()

	var out []domain.Subscription
	for rows.Next() {
		s, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *SubscriptionRepository) ListAllActive(ctx context.Context) ([]domain.Subscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+subscriptionCols+`
		FROM saas.subscriptions WHERE status='active' ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("listar suscripciones activas: %w", err)
	}
	defer rows.Close()

	var out []domain.Subscription
	for rows.Next() {
		s, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}
