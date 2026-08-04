package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
)

// Add y ListByPurchase implementan domain.WithholdingRepository sobre el mismo *Repository
// (mismo pool) que ya implementa domain.Repository — separado en su propio archivo por
// claridad, mismo patrón que payment_repository.go.

func (r *Repository) Add(ctx context.Context, w domain.PurchaseWithholding) (*domain.PurchaseWithholding, error) {
	w.ID = uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO purchase.purchase_withholdings
			(id, purchase_order_id, concept_code, concept_name, base, rate_bp, amount, account_payable)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		w.ID, w.PurchaseOrderID, w.ConceptCode, w.ConceptName, w.Base, w.RateBP, w.Amount, w.AccountPayable,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar retención: %w", err)
	}
	return &w, nil
}

func (r *Repository) ListByPurchase(ctx context.Context, purchaseOrderID uuid.UUID) ([]domain.PurchaseWithholding, error) {
	return r.loadWithholdings(ctx, purchaseOrderID)
}

func (r *Repository) GetWithholdingSummary(ctx context.Context, companyID, supplierID uuid.UUID, year int) ([]domain.WithholdingSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT w.concept_code, w.concept_name, SUM(w.base), SUM(w.amount)
		FROM purchase.purchase_withholdings w
		JOIN purchase.orders o ON o.id = w.purchase_order_id
		WHERE o.company_id = $1 AND o.supplier_id = $2
		  AND EXTRACT(YEAR FROM o.issue_date) = $3
		GROUP BY w.concept_code, w.concept_name
		ORDER BY w.concept_code`,
		companyID, supplierID, year,
	)
	if err != nil {
		return nil, fmt.Errorf("resumen de retenciones: %w", err)
	}
	defer rows.Close()

	var out []domain.WithholdingSummary
	for rows.Next() {
		var s domain.WithholdingSummary
		if err := rows.Scan(&s.ConceptCode, &s.ConceptName, &s.Base, &s.Amount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
