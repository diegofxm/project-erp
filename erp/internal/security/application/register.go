package application

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/diegofxm/erp/internal/security/domain"
)

type RegisterUseCase struct {
	repo  domain.Repository
	token domain.TokenService
}

func NewRegisterUseCase(repo domain.Repository, token domain.TokenService) *RegisterUseCase {
	return &RegisterUseCase{repo: repo, token: token}
}

type RegisterRequest struct {
	Email    string
	Password string
	Name     string
}

func (uc *RegisterUseCase) Execute(ctx context.Context, req RegisterRequest) (*domain.AuthResult, error) {
	if _, err := uc.repo.GetByEmail(ctx, req.Email); err == nil {
		return nil, domain.ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashStr := string(hash)

	u := domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: &hashStr,
		Name:         req.Name,
		Role:         domain.RoleAdmin,
		IsActive:     true,
	}

	saved, err := uc.repo.Save(ctx, u)
	if err != nil {
		return nil, err
	}

	tok, err := uc.token.Issue(saved.ID, uuid.Nil, "", saved.IsSuperAdmin, saved.TokenVersion)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResult{User: *saved, Token: tok}, nil
}
