package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/diegofxm/api-dian/internal/issuers"
)

// Service centraliza el registro/login de usuarios. "Un usuario = un emisor": Register crea
// el emisor Y su primer usuario admin en una sola llamada (flujo típico de alta de un SaaS),
// en vez de exponer la creación de emisores como un endpoint público separado.
type Service struct {
	repo    Repository
	issuers IssuerPort
	tokens  *TokenIssuer
}

// New crea el servicio de autenticación.
func New(repo Repository, issuerPort IssuerPort, tokens *TokenIssuer) *Service {
	return &Service{repo: repo, issuers: issuerPort, tokens: tokens}
}

// RegisterRequest agrupa los datos del emisor nuevo y de su primer usuario admin.
type RegisterRequest struct {
	Issuer   issuers.Issuer
	Email    string
	Password string
	Name     string
}

// AuthResult es lo que Register/Login devuelven: el usuario y un token de acceso ya firmado.
type AuthResult struct {
	User  User
	Token string
}

// Register crea el emisor y su primer usuario admin. Valida la disponibilidad del correo
// ANTES de crear el emisor — si se creara el emisor primero y luego el correo resultara
// duplicado, quedaría un emisor "huérfano" sin ningún usuario que lo administre. Esta
// comprobación cubre el caso normal (correo repetido); la restricción UNIQUE en la base de
// datos sigue siendo la red de seguridad real contra una carrera entre dos registros
// simultáneos con el mismo correo.
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

	iss, err := s.issuers.RegisterIssuer(ctx, req.Issuer)
	if err != nil {
		return nil, err
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("cifrar contraseña: %w", err)
	}

	u, err := s.repo.Create(ctx, User{
		IssuerID:     iss.ID,
		Email:        email,
		PasswordHash: hash,
		Name:         strings.TrimSpace(req.Name),
		Role:         RoleAdmin,
		IsActive:     true,
	})
	if err != nil {
		return nil, err
	}

	return s.issueResult(u)
}

// Login valida credenciales y devuelve un token nuevo.
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

	return s.issueResult(u)
}

func (s *Service) issueResult(u *User) (*AuthResult, error) {
	token, err := s.tokens.Issue(*u)
	if err != nil {
		return nil, fmt.Errorf("emitir token: %w", err)
	}
	return &AuthResult{User: *u, Token: token}, nil
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
