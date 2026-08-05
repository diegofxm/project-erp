// Package electronic implementa saas/domain.DocumentCounterPort — cuenta documentos electrónicos
// emitidos (status != draft) directamente contra electronic.documents. Consulta SQL directa en
// vez de pasar por electronic/domain.DocumentRepository.ListByCompany porque ese método está
// pensado para paginar resultados completos (Limit=0 devolvería cero filas), no para contar.
package electronic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Adapter struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Adapter {
	return &Adapter{pool: pool}
}

func (a *Adapter) CountInPeriod(ctx context.Context, companyID uuid.UUID, from, to time.Time) (int, error) {
	var count int
	err := a.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM electronic.documents
		WHERE company_id = $1 AND status != 'draft' AND created_at >= $2 AND created_at < $3`,
		companyID, from, to,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("contar documentos del período: %w", err)
	}
	return count, nil
}
