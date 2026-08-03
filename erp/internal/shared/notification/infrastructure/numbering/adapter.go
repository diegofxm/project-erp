// Package numbering implementa notification/domain.NumberingPort leyendo desde
// el repositorio de rangos de numeración de electronic.
package numbering

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	electronicdomain "github.com/diegofxm/erp/internal/electronic/domain"
	"github.com/diegofxm/erp/internal/shared/notification/domain"
)

// Adapter implementa domain.NumberingPort usando el repositorio de rangos de electronic.
type Adapter struct {
	repo electronicdomain.NumberingRepository
}

// New crea el adaptador.
func New(repo electronicdomain.NumberingRepository) *Adapter {
	return &Adapter{repo: repo}
}

var _ domain.NumberingPort = (*Adapter)(nil)

// ListByCompany trae todos los tipos de documento ("" = sin filtro) — la
// campana avisa de cualquier resolución de la empresa, no de un tipo puntual.
func (a *Adapter) ListByCompany(ctx context.Context, companyID uuid.UUID) ([]domain.RangeInfo, error) {
	ranges, err := a.repo.ListByCompany(ctx, companyID, "")
	if err != nil {
		return nil, fmt.Errorf("numbering adapter: %w", err)
	}
	out := make([]domain.RangeInfo, len(ranges))
	for i, r := range ranges {
		out[i] = domain.RangeInfo{
			ID:                   r.ID,
			DianDocumentTypeCode: r.DianDocumentTypeCode,
			Prefix:               r.Prefix,
			Status:               r.Status(),
			ValidTo:              r.ValidTo,
		}
	}
	return out, nil
}
