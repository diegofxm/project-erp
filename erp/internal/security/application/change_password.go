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
	repo domain.Repository
}

func NewChangePasswordUseCase(repo domain.Repository) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{repo: repo}
}

func (uc *ChangePasswordUseCase) Execute(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	u, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(currentPassword)) != nil {
		return domain.ErrCurrentPasswordIncorrect
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return uc.repo.UpdatePassword(ctx, userID, string(hash))
}
