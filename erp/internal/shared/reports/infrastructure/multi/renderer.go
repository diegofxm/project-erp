// Package multi implementa domain.Renderer enrutando al generador correcto según el formato.
// Es la implementación que se inyecta en main.go: un único Renderer para todos los formatos.
package multi

import (
	"context"
	"fmt"

	"github.com/diegofxm/erp/internal/shared/reports/domain"
)

type Renderer struct {
	renderers map[domain.Format]domain.Renderer
}

func New(renderers map[domain.Format]domain.Renderer) *Renderer {
	return &Renderer{renderers: renderers}
}

var _ domain.Renderer = (*Renderer)(nil)

func (r *Renderer) Render(ctx context.Context, req domain.RenderRequest) ([]byte, error) {
	impl, ok := r.renderers[req.Format]
	if !ok {
		return nil, fmt.Errorf("reports: formato no registrado: %s", req.Format)
	}
	return impl.Render(ctx, req)
}
