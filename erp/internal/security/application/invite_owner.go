package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/security/domain"
)

// InviteOwnerUseCase crea un usuario invitado SIN empresa todavía — a diferencia de
// InviteUserUseCase (que vincula de una vez a una empresa existente con rol admin/member), este
// caso es para cuando la empresa ni siquiera existe aún (ej. aprobar un prospecto nuevo, ver
// saas/application/prospect.go): primero se crea el usuario, y quien lo cree después usa su ID
// como creatorID de company.CreateUseCase, que lo vincula como "owner" automáticamente.
type InviteOwnerUseCase struct {
	repo domain.Repository
}

func NewInviteOwnerUseCase(repo domain.Repository) *InviteOwnerUseCase {
	return &InviteOwnerUseCase{repo: repo}
}

// Execute es idempotente por correo: si ya existe un usuario con ese correo que todavía no
// aceptó la invitación (sin password) y no tiene ninguna empresa vinculada, reutiliza esa fila en
// vez de fallar — así reintentar una aprobación de prospecto que falló a mitad de camino (ej. la
// empresa no se pudo crear) no queda bloqueada por "correo ya existe".
func (uc *InviteOwnerUseCase) Execute(ctx context.Context, email, name string) (*domain.User, string, error) {
	if existing, err := uc.repo.GetByEmail(ctx, email); err == nil {
		if existing.PasswordHash != nil {
			return nil, "", domain.ErrEmailTaken
		}
		companies, err := uc.repo.ListCompanyIDs(ctx, existing.ID)
		if err != nil {
			return nil, "", err
		}
		if len(companies) > 0 {
			return nil, "", domain.ErrEmailTaken
		}
		tok := uuid.New()
		expires := time.Now().Add(72 * time.Hour)
		if _, err := uc.repo.UpdateProfile(ctx, existing.ID, name, email); err != nil {
			return nil, "", err
		}
		if err := uc.repo.SetInviteToken(ctx, existing.ID, tok, expires); err != nil {
			return nil, "", err
		}
		return existing, tok.String(), nil
	}

	tok := uuid.New()
	expires := time.Now().Add(72 * time.Hour)
	u := domain.User{
		ID:                   uuid.New(),
		Email:                email,
		Name:                 name,
		Role:                 domain.RoleAdmin,
		IsActive:             true,
		InviteToken:          &tok,
		InviteTokenExpiresAt: &expires,
	}
	saved, err := uc.repo.Save(ctx, u)
	if err != nil {
		return nil, "", err
	}
	return saved, tok.String(), nil
}
