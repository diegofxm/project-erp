// Package security implementa saas/domain.UserPort leyendo del repositorio de usuarios del ERP.
package security

import (
	"context"

	"github.com/diegofxm/erp/internal/saas/domain"
	securitydomain "github.com/diegofxm/erp/internal/security/domain"
)

type Adapter struct {
	repo securitydomain.Repository
}

func New(repo securitydomain.Repository) *Adapter {
	return &Adapter{repo: repo}
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
