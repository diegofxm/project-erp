package application

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/diegofxm/erp/internal/security/domain"
)

type LoginUseCase struct {
	repo    domain.Repository
	token   domain.TokenService
	limiter *LoginRateLimiter
}

func NewLoginUseCase(repo domain.Repository, token domain.TokenService, limiter *LoginRateLimiter) *LoginUseCase {
	return &LoginUseCase{repo: repo, token: token, limiter: limiter}
}

func (uc *LoginUseCase) Execute(ctx context.Context, email, password string) (*domain.AuthResult, error) {
	key := strings.ToLower(strings.TrimSpace(email))
	if uc.limiter != nil && !uc.limiter.Allow(key) {
		return nil, domain.ErrTooManyAttempts
	}

	u, err := uc.repo.GetByEmail(ctx, email)
	if err != nil {
		if uc.limiter != nil {
			uc.limiter.RecordFailure(key)
		}
		return nil, domain.ErrInvalidPassword // no revelar si el correo existe
	}
	if !u.IsActive {
		return nil, domain.ErrUserInactive
	}
	if u.PasswordHash == nil {
		if uc.limiter != nil {
			uc.limiter.RecordFailure(key)
		}
		return nil, domain.ErrInvalidPassword
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		if uc.limiter != nil {
			uc.limiter.RecordFailure(key)
		}
		return nil, domain.ErrInvalidPassword
	}
	if uc.limiter != nil {
		uc.limiter.RecordSuccess(key)
	}

	// Auto-seleccionar empresa si el usuario tiene exactamente una
	companyIDs, _ := uc.repo.ListCompanyIDs(ctx, u.ID)
	var companyID uuid.UUID
	var role string
	if len(companyIDs) == 1 {
		companyID = companyIDs[0]
		role, _ = uc.repo.GetRole(ctx, u.ID, companyID)
	}

	tok, err := uc.token.Issue(u.ID, companyID, role, u.IsSuperAdmin, u.TokenVersion)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResult{User: *u, Token: tok, CompanyID: companyID, Role: role}, nil
}
