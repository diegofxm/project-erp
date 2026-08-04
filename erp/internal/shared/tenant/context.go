package tenant

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type companyKey struct{}
type userKey struct{}
type roleKey struct{}

// Verifier verifica un JWT y devuelve userID, companyID y el rol del usuario en esa empresa.
// Implementado por security/infrastructure/jwt.TokenService.
type Verifier interface {
	Verify(raw string) (userID, companyID uuid.UUID, role string, err error)
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

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey{}, role)
}

// GetRole devuelve el rol del usuario en la empresa activa ("owner"/"admin"/"member"), o "" si
// no hay sesión de empresa (ej. el usuario aún no seleccionó empresa).
func GetRole(ctx context.Context) string {
	role, _ := ctx.Value(roleKey{}).(string)
	return role
}

// CanManage indica si el rol de la sesión puede administrar la empresa (usuarios, credenciales,
// perfil, facturación) — hoy eso es owner y admin. member puede operar el resto de módulos
// (ventas, compras, inventario, contabilidad, etc.) pero no estas operaciones administrativas.
func CanManage(ctx context.Context) bool {
	switch GetRole(ctx) {
	case "owner", "admin":
		return true
	default:
		return false
	}
}

// Middleware extrae el JWT del header Authorization e inyecta userID, companyID y role en el
// contexto. Es silencioso si no hay token o es inválido — las rutas protegidas llaman
// requireAuth/requireTenant, y las que además requieren rol de administración llaman CanManage.
func Middleware(v Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if raw != "" {
				if userID, companyID, role, err := v.Verify(raw); err == nil {
					r = r.WithContext(WithUserID(r.Context(), userID))
					r = r.WithContext(WithCompanyID(r.Context(), companyID))
					r = r.WithContext(WithRole(r.Context(), role))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
