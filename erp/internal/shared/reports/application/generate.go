package application

import (
	"context"

	"github.com/diegofxm/erp/internal/shared/reports/domain"
)

// GenerateUseCase orquesta: cargar template → render HTML → exportar al formato pedido.
type GenerateUseCase struct {
	renderer domain.Renderer
}

func NewGenerateUseCase(renderer domain.Renderer) *GenerateUseCase {
	return &GenerateUseCase{renderer: renderer}
}

func (uc *GenerateUseCase) Generate(ctx context.Context, req domain.RenderRequest) ([]byte, error) {
	return uc.renderer.Render(ctx, req)
}
