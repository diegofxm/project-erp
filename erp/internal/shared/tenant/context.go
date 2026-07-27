package tenant

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey struct{}

// WithCompanyID inyecta el company_id en el contexto de la request.
func WithCompanyID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// GetCompanyID extrae el company_id del contexto. Devuelve uuid.Nil si no fue inyectado
// (nunca debería ocurrir en handlers protegidos — el middleware lo garantiza).
func GetCompanyID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(contextKey{}).(uuid.UUID)
	return id
}

// Middleware extrae el company_id del JWT (vía security/) y lo inyecta al contexto.
// Toda request a rutas protegidas pasa por aquí antes de llegar a cualquier handler.
// TODO: cuando security/ esté implementado, reemplazar este placeholder con la
// extracción real del claim "company_id" del token JWT.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// placeholder: sin JWT todavía
		next.ServeHTTP(w, r)
	})
}
