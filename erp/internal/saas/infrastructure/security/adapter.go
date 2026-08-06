// Package security implementa saas/domain.UserPort leyendo del repositorio de usuarios del ERP.
package security

import (
	"context"

	"github.com/google/uuid"

	securityapp "github.com/diegofxm/erp/internal/security/application"
	securitydomain "github.com/diegofxm/erp/internal/security/domain"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type Adapter struct {
	repo        securitydomain.Repository
	inviteOwner *securityapp.InviteOwnerUseCase
}

func New(repo securitydomain.Repository, inviteOwner *securityapp.InviteOwnerUseCase) *Adapter {
	return &Adapter{repo: repo, inviteOwner: inviteOwner}
}

var _ domain.UserPort = (*Adapter)(nil)

func (a *Adapter) ListAll(ctx context.Context) ([]domain.PlatformUser, error) {
	users, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PlatformUser, len(users))
	for i, u := range users {
		out[i] = domain.PlatformUser{
			ID: u.ID, Email: u.Email, Name: u.Name, Role: string(u.Role),
			IsSuperAdmin: u.IsSuperAdmin, IsActive: u.IsActive,
			InviteAcceptedAt: u.InviteAcceptedAt, CreatedAt: u.CreatedAt,
		}
	}
	return out, nil
}

func (a *Adapter) SetSuperAdmin(ctx context.Context, userID uuid.UUID, value bool) error {
	return a.repo.SetSuperAdmin(ctx, userID, value)
}

func (a *Adapter) InviteOwner(ctx context.Context, email, name string) (uuid.UUID, string, error) {
	u, token, err := a.inviteOwner.Execute(ctx, email, name)
	if err != nil {
		return uuid.Nil, "", err
	}
	return u.ID, token, nil
}
