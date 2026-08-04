package application

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/diegofxm/erp/internal/security/domain"
)

type LoginUseCase struct {
	repo  domain.Repository
	token domain.TokenService
}

func NewLoginUseCase(repo domain.Repository, token domain.TokenService) *LoginUseCase {
	return &LoginUseCase{repo: repo, token: token}
}

func (uc *LoginUseCase) Execute(ctx context.Context, email, password string) (*domain.AuthResult, error) {
	u, err := uc.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrInvalidPassword // no revelar si el correo existe
	}
	if !u.IsActive {
		return nil, domain.ErrUserInactive
	}
	if u.PasswordHash == nil {
		return nil, domain.ErrInvalidPassword
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrInvalidPassword
	}

	// Auto-seleccionar empresa si el usuario tiene exactamente una
	companyIDs, _ := uc.repo.ListCompanyIDs(ctx, u.ID)
	var companyID uuid.UUID
	var role string
	if len(companyIDs) == 1 {
		companyID = companyIDs[0]
		role, _ = uc.repo.GetRole(ctx, u.ID, companyID)
	}

	tok, err := uc.token.Issue(u.ID, companyID, role)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResult{User: *u, Token: tok, CompanyID: companyID}, nil
}
