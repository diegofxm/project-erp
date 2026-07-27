package tenant

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type companyKey struct{}
type userKey struct{}

// Verifier verifica un JWT y devuelve userID y companyID.
// Implementado por security/infrastructure/jwt.TokenService.
type Verifier interface {
	Verify(raw string) (userID, companyID uuid.UUID, err error)
}

func WithCompanyID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, companyKey{}, id)
}

func GetCompanyID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(companyKey{}).(uuid.UUID)
	return id
}

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userKey{}, id)
}

func GetUserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userKey{}).(uuid.UUID)
	return id
}

// Middleware extrae el JWT del header Authorization e inyecta userID y companyID en el contexto.
// Es silencioso si no hay token o es inválido — las rutas protegidas llaman requireAuth.
func Middleware(v Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if raw != "" {
				if userID, companyID, err := v.Verify(raw); err == nil {
					r = r.WithContext(WithUserID(r.Context(), userID))
					r = r.WithContext(WithCompanyID(r.Context(), companyID))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
