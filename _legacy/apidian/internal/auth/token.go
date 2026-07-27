package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenTTL es la vigencia del token de acceso. Sin refresh token todavía: un solo JWT firmado
// con expiración de 24h, proporcional al tamaño actual del proyecto (panel administrativo, no
// banca). Si más adelante hace falta revocación inmediata, el camino natural es agregar
// refresh tokens guardados en BD (revocables), no reemplazar esto.
const TokenTTL = 24 * time.Hour

// claims usa nombres propios en vez de los estándar de JWT para evitar la confusión de
// terminología entre "Issuer" del documento UBL/DIAN (quien emite la factura) e "iss" del
// propio token JWT (quien firmó el token, que aquí siempre es apidian). TenantID es la
// empresa ACTIVA en este token — puede ser uuid.Nil (ver Issue).
type claims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	jwt.RegisteredClaims
}

// TokenIssuer firma y valida los JWT de sesión con un secreto compartido (HS256).
type TokenIssuer struct {
	secret []byte
}

// NewTokenIssuer crea un TokenIssuer. secret es AUTH_JWT_SECRET, nunca la misma clave que
// ISSUER_SECRETS_KEY (cifra datos en reposo, no firma tokens) — ver internal/config.
func NewTokenIssuer(secret []byte) *TokenIssuer {
	return &TokenIssuer{secret: secret}
}

// Issue firma un nuevo token de acceso para u. tenantID es la empresa ACTIVA en este token —
// uuid.Nil si el usuario todavía no tiene ninguna empresa vinculada, o tiene varias y no ha
// seleccionado cuál usar (ver auth.Service.Login/SelectIssuer, sección 9.32 del architecture
// doc). Ya no se lee de u.IssuerID: cuál empresa está activa es contextual, no una propiedad
// fija del usuario.
func (t *TokenIssuer) Issue(u User, tenantID uuid.UUID) (string, error) {
	now := time.Now()
	c := claims{
		UserID:   u.ID,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "apidian",
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(t.secret)
}

// Verify valida la firma y expiración de raw y devuelve el UserID/TenantID embebidos.
func (t *TokenIssuer) Verify(raw string) (userID, tenantID uuid.UUID, err error) {
	var c claims
	_, err = jwt.ParseWithClaims(raw, &c, func(*jwt.Token) (interface{}, error) {
		return t.secret, nil
	})
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidToken
	}
	return c.UserID, c.TenantID, nil
}
