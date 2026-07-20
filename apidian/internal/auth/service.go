package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/google/uuid"
)

// Service centraliza el registro/login de usuarios y su acceso a empresas (user_issuers).
// Desde la Fase 9.32: registro y creación de empresa están desacoplados — un usuario puede
// existir sin ninguna empresa vinculada, y puede crear/vincularse a varias (multi-empresa/
// sucursales) — ver docs/apidian-architecture.md sección 9.32.
type Service struct {
	repo    Repository
	issuers IssuerPort
	tokens  *TokenIssuer
}

// New crea el servicio de autenticación.
func New(repo Repository, issuerPort IssuerPort, tokens *TokenIssuer) *Service {
	return &Service{repo: repo, issuers: issuerPort, tokens: tokens}
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
