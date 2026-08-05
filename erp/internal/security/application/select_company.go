package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/security/domain"
)

type SelectCompanyUseCase struct {
	repo  domain.Repository
	token domain.TokenService
}

func NewSelectCompanyUseCase(repo domain.Repository, token domain.TokenService) *SelectCompanyUseCase {
	return &SelectCompanyUseCase{repo: repo, token: token}
}

func (uc *SelectCompanyUseCase) Execute(ctx context.Context, userID, companyID uuid.UUID) (*domain.AuthResult, error) {
	u, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	ok, err := uc.repo.HasCompany(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrNotAMember
	}

	role, err := uc.repo.GetRole(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}

	tok, err := uc.token.Issue(userID, companyID, role, u.IsSuperAdmin)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResult{User: *u, Token: tok, CompanyID: companyID}, nil
}
