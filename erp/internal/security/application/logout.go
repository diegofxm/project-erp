package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/security/domain"
)

// LogoutUseCase revoca la sesión actual (y, de paso, cualquier otra) del lado del servidor --
// antes "cerrar sesión" solo podía borrar el token del lado del cliente; el token en sí seguía
// siendo válido hasta su expiración (24h) si alguien más llegaba a tenerlo. Incrementar
// token_version invalida TODAS las sesiones del usuario, no solo la del dispositivo que pide el
// logout -- no hay hoy una tabla de sesiones por dispositivo/token individual, así que "cerrar
// esta sesión" y "cerrar sesión en todos lados" son, por ahora, la misma operación.
type LogoutUseCase struct {
	repo domain.Repository
}

func NewLogoutUseCase(repo domain.Repository) *LogoutUseCase {
	return &LogoutUseCase{repo: repo}
}

func (uc *LogoutUseCase) Execute(ctx context.Context, userID uuid.UUID) error {
	return uc.repo.IncrementTokenVersion(ctx, userID)
}
