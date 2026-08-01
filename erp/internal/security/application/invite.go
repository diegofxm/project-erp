package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/diegofxm/erp/internal/security/domain"
)

// InviteUserUseCase crea un usuario sin contraseña y genera un token de invitación (72h).
type InviteUserUseCase struct {
	repo domain.Repository
}

func NewInviteUserUseCase(repo domain.Repository) *InviteUserUseCase {
	return &InviteUserUseCase{repo: repo}
}

func (uc *InviteUserUseCase) Execute(ctx context.Context, email, name string) (*domain.User, error) {
	if _, err := uc.repo.GetByEmail(ctx, email); err == nil {
		return nil, domain.ErrEmailTaken
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

	return uc.repo.Save(ctx, u)
	// TODO: emitir evento para que shared/notification envíe el correo de invitación
}

// AcceptInviteUseCase valida el token, establece la contraseña y devuelve una sesión.
type AcceptInviteUseCase struct {
	repo  domain.Repository
	token domain.TokenService
}

func NewAcceptInviteUseCase(repo domain.Repository, token domain.TokenService) *AcceptInviteUseCase {
	return &AcceptInviteUseCase{repo: repo, token: token}
}

func (uc *AcceptInviteUseCase) Execute(ctx context.Context, rawToken, password string) (*domain.AuthResult, error) {
	tok, err := uuid.Parse(rawToken)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	u, err := uc.repo.GetByInviteToken(ctx, tok)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if u.InviteTokenExpiresAt == nil || time.Now().After(*u.InviteTokenExpiresAt) {
		return nil, domain.ErrInvalidToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.SetPassword(ctx, u.ID, string(hash)); err != nil {
		return nil, err
	}

	updated, err := uc.repo.GetByID(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	jwtTok, err := uc.token.Issue(updated.ID, uuid.Nil)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResult{User: *updated, Token: jwtTok}, nil
}
