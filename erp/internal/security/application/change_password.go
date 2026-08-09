package application

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/diegofxm/erp/internal/security/domain"
)

// ChangePasswordUseCase cambia la contraseña de un usuario ya autenticado — exige la contraseña
// actual (a diferencia de AcceptInviteUseCase, que la establece por primera vez con solo el
// token de invitación como prueba de identidad).
type ChangePasswordUseCase struct {
	repo  domain.Repository
	token domain.TokenService
}

func NewChangePasswordUseCase(repo domain.Repository, token domain.TokenService) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{repo: repo, token: token}
}

// Execute cambia la contraseña y revoca todas las sesiones existentes (ver
// Repository.IncrementTokenVersion) -- en principio, incluida la sesión actual. Para no
// cerrarle la sesión al propio usuario que acaba de demostrar dos veces quién es (contraseña
// actual + nueva), se re-emite un token fresco ya con el token_version incrementado y se
// devuelve, para que el llamador lo guarde de inmediato. Cualquier OTRA sesión activa (otro
// dispositivo, otra pestaña con un token viejo) sí queda inválida en su siguiente request.
func (uc *ChangePasswordUseCase) Execute(ctx context.Context, userID, companyID uuid.UUID, role, currentPassword, newPassword string) (string, error) {
	u, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if u.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(currentPassword)) != nil {
		return "", domain.ErrCurrentPasswordIncorrect
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	if err := uc.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return "", err
	}
	if err := uc.repo.IncrementTokenVersion(ctx, userID); err != nil {
		return "", err
	}
	return uc.token.Issue(userID, companyID, role, u.IsSuperAdmin, u.TokenVersion+1)
}
