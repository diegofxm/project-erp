package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/security/domain"
)

// SessionVerifier implementa shared/tenant.Verifier. A diferencia de domain.TokenService (pura
// criptografía, sin acceso a BD), esta capa sí consulta el usuario real para comparar el
// token_version del JWT contra el valor actual en BD -- así una sesión revocada (logout, cambio
// de contraseña) deja de ser válida en el siguiente request aunque la firma y expiración del
// JWT en sí sigan siendo correctas. Es la pieza que hace posible el punto 14 del plan de acción
// 2026-08-09: sin esto, "logout"/"invalidar sesión" solo podían borrar el token del lado del
// cliente, sin ningún efecto real del lado del servidor.
type SessionVerifier struct {
	tokens domain.TokenService
	repo   domain.Repository
}

func NewSessionVerifier(tokens domain.TokenService, repo domain.Repository) *SessionVerifier {
	return &SessionVerifier{tokens: tokens, repo: repo}
}

func (v *SessionVerifier) Verify(ctx context.Context, raw string) (uuid.UUID, uuid.UUID, string, bool, error) {
	userID, companyID, role, isSuperAdmin, tokenVersion, err := v.tokens.Verify(raw)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", false, err
	}
	u, err := v.repo.GetByID(ctx, userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", false, fmt.Errorf("usuario no encontrado")
	}
	if !u.IsActive {
		return uuid.Nil, uuid.Nil, "", false, fmt.Errorf("cuenta inactiva")
	}
	if u.TokenVersion != tokenVersion {
		return uuid.Nil, uuid.Nil, "", false, fmt.Errorf("sesión revocada")
	}
	return userID, companyID, role, isSuperAdmin, nil
}
