package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/diegofxm/apidian/internal/email"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/google/uuid"
)

// Service centraliza el registro/login de usuarios y su acceso a empresas (user_issuers).
// Desde la Fase 9.32: registro y creación de empresa están desacoplados — un usuario puede
// existir sin ninguna empresa vinculada, y puede crear/vincularse a varias (multi-empresa/
// sucursales) — ver docs/apidian-architecture.md sección 9.32.
type Service struct {
	repo       Repository
	issuers    IssuerPort
	tokens     *TokenIssuer
	emailSvc   EmailPort
	appBaseURL string
}

// New crea el servicio de autenticación.
func New(repo Repository, issuerPort IssuerPort, tokens *TokenIssuer) *Service {
	return &Service{repo: repo, issuers: issuerPort, tokens: tokens}
}

// WithEmail habilita el envío de correos de invitación. appBaseURL es la URL raíz del
// frontend (ej. https://app.cofacture.co) — se usa para construir el link del correo.
func (s *Service) WithEmail(sender EmailPort, appBaseURL string) *Service {
	s.emailSvc = sender
	s.appBaseURL = appBaseURL
	return s
}

// RegisterRequest son los datos del usuario nuevo — ya NO incluye una empresa (eso es
// CreateIssuerForUser, después de registrarse).
type RegisterRequest struct {
	Email    string
	Password string
	Name     string
}

// AuthResult es lo que Register/Login/CreateIssuerForUser/SelectIssuer devuelven: el usuario,
// un token de acceso ya firmado, y la empresa activa en ese token (nil si todavía no tiene
// ninguna o no se seleccionó cuál usar entre varias).
type AuthResult struct {
	User         User
	Token        string
	ActiveIssuer *issuers.Issuer
}

// Register crea SOLO el usuario — sin empresa. El siguiente paso natural es
// CreateIssuerForUser (crear la primera) o, si un administrador ya lo vinculó a una empresa
// existente, SelectIssuer.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResult, error) {
	if err := validateCredentials(req.Email, req.Password, req.Name); err != nil {
		return nil, err
	}
	email := normalizeEmail(req.Email)

	if _, err := s.repo.GetByEmail(ctx, email); err == nil {
		return nil, ErrEmailAlreadyExists
	} else if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("cifrar contraseña: %w", err)
	}

	u, err := s.repo.Create(ctx, User{
		Email:        email,
		PasswordHash: hash,
		Name:         strings.TrimSpace(req.Name),
		Role:         RoleAdmin,
		IsActive:     true,
	})
	if err != nil {
		return nil, err
	}

	// Usuario recién creado: cero empresas vinculadas todavía, ninguna activa.
	return s.issueResult(u, nil)
}

// Login valida credenciales y devuelve un token nuevo. La empresa activa se decide así: si el
// usuario tiene EXACTAMENTE una empresa vinculada, se autoselecciona (mismo comportamiento
// de siempre para el caso normal de hoy); si tiene cero o varias, el token queda sin empresa
// activa — el cliente debe crear la primera (POST /issuers) o elegir entre las que tiene
// (POST /issuers/{id}/select).
func (s *Service) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	u, err := s.repo.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !u.IsActive {
		return nil, ErrUserInactive
	}
	if !checkPassword(u.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	issuerIDs, err := s.repo.ListIssuerIDs(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	var active *issuers.Issuer
	if len(issuerIDs) == 1 {
		active, err = s.issuers.GetIssuer(ctx, issuerIDs[0])
		if err != nil {
			return nil, err
		}
	}
	return s.issueResult(u, active)
}

// CreateIssuerForUser crea una empresa nueva y la vincula a userID como "owner" — el camino
// para completar el "espacio de empresa" después de registrarse, o para agregar una empresa/
// sucursal adicional a un usuario que ya tiene otras. La empresa recién creada queda activa
// en el token devuelto (tiene sentido operar de inmediato en lo que se acaba de crear, sin un
// paso extra de selección).
func (s *Service) CreateIssuerForUser(ctx context.Context, userID uuid.UUID, iss issuers.Issuer) (*AuthResult, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	created, err := s.issuers.RegisterIssuer(ctx, iss)
	if err != nil {
		return nil, err
	}

	if err := s.repo.LinkIssuer(ctx, userID, created.ID, RoleOwner); err != nil {
		return nil, err
	}

	return s.issueResult(u, created)
}

// ListUserIssuers devuelve las empresas a las que userID tiene acceso — para que el cliente
// muestre un selector ("mis empresas") cuando hay más de una.
func (s *Service) ListUserIssuers(ctx context.Context, userID uuid.UUID) ([]*issuers.Issuer, error) {
	ids, err := s.repo.ListIssuerIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*issuers.Issuer, 0, len(ids))
	for _, id := range ids {
		iss, err := s.issuers.GetIssuer(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, iss)
	}
	return out, nil
}

// SelectIssuer reemite el token con issuerID como empresa activa — solo si userID de verdad
// tiene acceso a ella (ErrIssuerAccessDenied si no, nunca confiar en que el cliente solo pide
// IDs a los que realmente tiene derecho).
func (s *Service) SelectIssuer(ctx context.Context, userID, issuerID uuid.UUID) (*AuthResult, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	ok, err := s.repo.HasAccess(ctx, userID, issuerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrIssuerAccessDenied
	}

	iss, err := s.issuers.GetIssuer(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	return s.issueResult(u, iss)
}

// IsSuperAdmin devuelve si el usuario es superadministrador del sistema.
func (s *Service) IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return u.IsSuperAdmin, nil
}

// UpdateProfile actualiza el nombre y/o correo del usuario autenticado. No toca contraseña ni rol.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, name, email string) (*User, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	if name == "" {
		return nil, ErrEmptyName
	}
	if email == "" {
		return nil, ErrEmptyEmail
	}
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.Name = name
	u.Email = email
	return s.repo.Update(ctx, *u)
}

func (s *Service) issueResult(u *User, activeIssuer *issuers.Issuer) (*AuthResult, error) {
	tenantID := uuid.Nil
	if activeIssuer != nil {
		tenantID = activeIssuer.ID
	}
	token, err := s.tokens.Issue(*u, tenantID)
	if err != nil {
		return nil, fmt.Errorf("emitir token: %w", err)
	}
	return &AuthResult{User: *u, Token: token, ActiveIssuer: activeIssuer}, nil
}

// ListUsers devuelve todos los usuarios del sistema — solo para el panel de superadmin.
func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	return s.repo.ListAll(ctx)
}

// CreateInvited crea un usuario sin contraseña y le envía un correo de invitación con un link
// mágico de un solo uso. El usuario completa su cuenta al hacer clic y configurar su contraseña
// (ver AcceptInvite). Requiere que WithEmail haya sido llamado; retorna ErrNoEmailSender si no.
func (s *Service) CreateInvited(ctx context.Context, userEmail, name string) (*User, error) {
	if s.emailSvc == nil {
		return nil, ErrNoEmailSender
	}
	userEmail = normalizeEmail(userEmail)
	name = strings.TrimSpace(name)
	if userEmail == "" {
		return nil, ErrEmptyEmail
	}
	if name == "" {
		return nil, ErrEmptyName
	}

	if _, err := s.repo.GetByEmail(ctx, userEmail); err == nil {
		return nil, ErrEmailAlreadyExists
	} else if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	token := uuid.New()
	expiry := time.Now().UTC().Add(72 * time.Hour)
	u, err := s.repo.Create(ctx, User{
		Email:                userEmail,
		Name:                 name,
		Role:                 RoleAdmin,
		IsActive:             true,
		InviteToken:          &token,
		InviteTokenExpiresAt: &expiry,
	})
	if err != nil {
		return nil, err
	}

	link := s.appBaseURL + "/setup-password?token=" + token.String()
	msg := email.Message{
		To:      userEmail,
		Subject: "Bienvenido a Cofacture — configura tu contraseña",
		BodyText: fmt.Sprintf(
			"Hola %s,\n\nEl administrador de Cofacture ha creado tu cuenta. "+
				"Haz clic en el siguiente link para configurar tu contraseña "+
				"(válido por 72 horas):\n\n%s\n\nSi no solicitaste esta cuenta, ignora este correo.",
			name, link,
		),
		BodyHTML: fmt.Sprintf(`<!DOCTYPE html>
<html lang="es"><body style="font-family:sans-serif;color:#1a1a1a;max-width:480px;margin:auto;padding:24px">
<h2 style="color:#14345C">Bienvenido a Cofacture</h2>
<p>Hola <strong>%s</strong>,</p>
<p>El administrador ha creado tu cuenta de facturación electrónica.
Haz clic en el botón para configurar tu contraseña:</p>
<p style="text-align:center;margin:32px 0">
  <a href="%s" style="background:#14345C;color:#fff;padding:12px 28px;border-radius:6px;text-decoration:none;font-weight:600">
    Configurar contraseña
  </a>
</p>
<p style="color:#666;font-size:13px">El link es válido por 72 horas. Si no puedes hacer clic,
copia y pega esta URL: <br><a href="%s">%s</a></p>
<p style="color:#999;font-size:12px">Si no solicitaste esta cuenta, ignora este correo.</p>
</body></html>`, name, link, link, link),
	}
	if err := s.emailSvc.Send(ctx, msg); err != nil {
		return nil, fmt.Errorf("enviar correo de invitación: %w", err)
	}
	return u, nil
}

// AcceptInvite valida el token de invitación, establece la contraseña y devuelve un token de
// sesión listo para usar — el usuario queda autenticado de inmediato tras configurar su cuenta.
func (s *Service) AcceptInvite(ctx context.Context, rawToken, password string) (*AuthResult, error) {
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	token, err := uuid.Parse(rawToken)
	if err != nil {
		return nil, ErrInvalidInviteToken
	}

	u, err := s.repo.GetByInviteToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidInviteToken
		}
		return nil, err
	}
	if u.InviteAcceptedAt != nil {
		return nil, ErrInviteAlreadyUsed
	}
	if u.InviteTokenExpiresAt != nil && time.Now().UTC().After(*u.InviteTokenExpiresAt) {
		return nil, ErrInviteTokenExpired
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("cifrar contraseña: %w", err)
	}
	if err := s.repo.SetPassword(ctx, u.ID, hash); err != nil {
		return nil, err
	}

	// Releer el usuario con el estado actualizado para emitir el token correcto.
	updated, err := s.repo.GetByID(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return s.issueResult(updated, nil)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateCredentials(email, password, name string) error {
	if strings.TrimSpace(email) == "" {
		return ErrEmptyEmail
	}
	if password == "" {
		return ErrEmptyPassword
	}
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	if strings.TrimSpace(name) == "" {
		return ErrEmptyName
	}
	return nil
}
