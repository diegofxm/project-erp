package jwt

import (
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type claims struct {
	UserID       uuid.UUID `json:"uid"`
	CompanyID    uuid.UUID `json:"cid"`
	Role         string    `json:"rol"`
	IsSuperAdmin bool      `json:"isa"`
	TokenVersion int       `json:"tv"`
	gojwt.RegisteredClaims
}

// TokenService implementa domain.TokenService con HMAC-SHA256, TTL 24h.
type TokenService struct {
	secret []byte
}

func NewTokenService(secret []byte) *TokenService {
	return &TokenService{secret: secret}
}

func (s *TokenService) Issue(userID, companyID uuid.UUID, role string, isSuperAdmin bool, tokenVersion int) (string, error) {
	c := claims{
		UserID:       userID,
		CompanyID:    companyID,
		Role:         role,
		IsSuperAdmin: isSuperAdmin,
		TokenVersion: tokenVersion,
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  gojwt.NewNumericDate(time.Now()),
		},
	}
	return gojwt.NewWithClaims(gojwt.SigningMethodHS256, c).SignedString(s.secret)
}

func (s *TokenService) Verify(raw string) (uuid.UUID, uuid.UUID, string, bool, int, error) {
	tok, err := gojwt.ParseWithClaims(raw, &claims{}, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil || !tok.Valid {
		return uuid.Nil, uuid.Nil, "", false, 0, fmt.Errorf("token inválido")
	}
	c, ok := tok.Claims.(*claims)
	if !ok {
		return uuid.Nil, uuid.Nil, "", false, 0, fmt.Errorf("claims inválidos")
	}
	return c.UserID, c.CompanyID, c.Role, c.IsSuperAdmin, c.TokenVersion, nil
}
