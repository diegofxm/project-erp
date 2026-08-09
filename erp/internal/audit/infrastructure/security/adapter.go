// Package security implementa audit/application.UserLookupPort envolviendo security.Repository
// -- mismo patrón de puerto local que ya usan sales/infrastructure/thirdparty,
// electronic/infrastructure/company, etc. Evita que audit/ importe security/domain directamente
// más allá de este único archivo de traducción.
package security

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/audit/application"
	securitydomain "github.com/diegofxm/erp/internal/security/domain"
)

// Repo es la vista mínima que este adaptador necesita de security.Repository.
type Repo interface {
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]securitydomain.User, error)
}

type Adapter struct {
	repo Repo
}

func New(repo Repo) *Adapter {
	return &Adapter{repo: repo}
}

func (a *Adapter) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]application.UserInfo, error) {
	users, err := a.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]application.UserInfo, len(users))
	for i, u := range users {
		out[i] = application.UserInfo{ID: u.ID, Email: u.Email, Name: u.Name}
	}
	return out, nil
}
