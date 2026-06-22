package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type authContextKey string

const (
	userIDKey   authContextKey = "auth_user_id"
	tenantIDKey authContextKey = "auth_tenant_id"
)

// TokenVerifier es lo mínimo que este middleware necesita de internal/auth — una interfaz
// angosta para que internal/api/middleware no dependa de internal/auth directamente.
type TokenVerifier interface {
	Verify(token string) (userID, tenantID uuid.UUID, err error)
}

// Auth exige un header "Authorization: Bearer <token>" válido y agrega el UserID/TenantID
// autenticados al contexto. Los handlers protegidos los leen con GetUserID/GetTenantID en vez
// de confiar en cualquier issuer_id que venga del cliente en el body o el path — así un
// usuario nunca puede operar sobre el emisor de otro.
func Auth(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				writeUnauthorized(w, "falta el header Authorization: Bearer <token>")
				return
			}

			userID, tenantID, err := verifier.Verify(token)
			if err != nil {
				writeUnauthorized(w, "token inválido o expirado")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			ctx = context.WithValue(ctx, tenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// GetUserID devuelve el ID del usuario autenticado — solo válido detrás de Auth().
func GetUserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDKey).(uuid.UUID)
	return id
}

// GetTenantID devuelve el ID del emisor (tenant DIAN) del usuario autenticado — solo válido
// detrás de Auth().
func GetTenantID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(tenantIDKey).(uuid.UUID)
	return id
}
