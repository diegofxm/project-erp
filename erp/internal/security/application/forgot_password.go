package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/diegofxm/erp/internal/security/domain"
	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
)

// resetTokenTTL -- 1 hora. Más corto que las 72h de un invite_token a propósito: acá el dueño de
// la cuenta ya existe y normalmente revisa su correo enseguida; una ventana larga solo alarga el
// tiempo en que un correo interceptado serviría para tomar la cuenta.
const resetTokenTTL = 1 * time.Hour

// ForgotPasswordUseCase genera un enlace de recuperación y lo envía por correo -- nunca revela si
// el correo existe en el sistema (mismo criterio que LoginUseCase con ErrInvalidPassword): tanto
// si el correo no existe como si el envío fue exitoso, Execute devuelve nil. Solo un error real
// de infraestructura (BD caída, proveedor de correo caído) se propaga.
type ForgotPasswordUseCase struct {
	repo     domain.Repository
	notifier notificationdomain.Notifier
	appURL   string
	limiter  *LoginRateLimiter
}

func NewForgotPasswordUseCase(repo domain.Repository, notifier notificationdomain.Notifier, appURL string, limiter *LoginRateLimiter) *ForgotPasswordUseCase {
	return &ForgotPasswordUseCase{repo: repo, notifier: notifier, appURL: appURL, limiter: limiter}
}

func (uc *ForgotPasswordUseCase) Execute(ctx context.Context, email string) error {
	key := strings.ToLower(strings.TrimSpace(email))
	// El límite cuenta CADA solicitud (no solo fallidas) -- a diferencia del login, acá hasta una
	// solicitud "exitosa" es una acción que un atacante puede repetir para bombardear la bandeja
	// de entrada de la víctima con correos de recuperación.
	if uc.limiter != nil {
		if !uc.limiter.Allow(key) {
			return domain.ErrTooManyAttempts
		}
		uc.limiter.RecordFailure(key)
	}

	u, err := uc.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil // no revelar si el correo existe
	}
	// Una cuenta con invitación pendiente (aún sin password) no tiene contraseña que recuperar --
	// reenviar la invitación sería lo correcto, pero eso es un flujo distinto (ver InviteUserUseCase).
	if u.PasswordHash == nil || !u.IsActive {
		return nil
	}

	tok := uuid.New()
	expires := time.Now().Add(resetTokenTTL)
	if err := uc.repo.SetResetToken(ctx, u.ID, tok, expires); err != nil {
		return err
	}

	if uc.notifier == nil {
		return nil
	}
	msg := notificationdomain.Message{
		To:         u.Email,
		Channel:    notificationdomain.ChannelEmail,
		Subject:    "Restablece tu contraseña",
		TemplateID: "password_reset",
		Data: map[string]any{
			"Name":      u.Name,
			"ResetURL":  uc.appURL + "/reset-password?token=" + tok.String(),
			"ExpiresIn": "1 hora",
		},
	}
	if err := uc.notifier.Send(ctx, msg); err != nil {
		return fmt.Errorf("enviar correo de recuperación: %w", err)
	}
	return nil
}

// ResetPasswordUseCase valida el token de recuperación, establece la nueva contraseña y revoca
// cualquier sesión existente -- a diferencia de AcceptInviteUseCase (que solo tiene que probar
// posesión del correo, porque la cuenta ni siquiera tenía contraseña todavía), acá SÍ había una
// contraseña antes, así que además de fijar la nueva hay que invalidar sesiones que pudieran
// seguir abiertas con la vieja.
type ResetPasswordUseCase struct {
	repo  domain.Repository
	token domain.TokenService
}

func NewResetPasswordUseCase(repo domain.Repository, token domain.TokenService) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{repo: repo, token: token}
}

func (uc *ResetPasswordUseCase) Execute(ctx context.Context, rawToken, newPassword string) (*domain.AuthResult, error) {
	tok, err := uuid.Parse(rawToken)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	u, err := uc.repo.GetByResetToken(ctx, tok)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if u.ResetTokenExpiresAt == nil || time.Now().After(*u.ResetTokenExpiresAt) {
		return nil, domain.ErrInvalidToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.UpdatePassword(ctx, u.ID, string(hash)); err != nil {
		return nil, err
	}
	if err := uc.repo.ClearResetToken(ctx, u.ID); err != nil {
		return nil, err
	}
	if err := uc.repo.IncrementTokenVersion(ctx, u.ID); err != nil {
		return nil, err
	}

	updated, err := uc.repo.GetByID(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	jwtTok, err := uc.token.Issue(updated.ID, uuid.Nil, "", updated.IsSuperAdmin, updated.TokenVersion)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResult{User: *updated, Token: jwtTok}, nil
}
