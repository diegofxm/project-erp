package logger

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/google/uuid"
)

type requestIDKey struct{}

// WithRequestID genera un ID corto por request (o reutiliza X-Request-Id si el cliente/proxy ya
// mandó uno) y lo expone tanto en el contexto (RequestID(ctx)) como en la cabecera de respuesta,
// para poder correlacionar una petición específica con sus líneas de log -- antes no había
// ninguna forma de hacer esto salvo por timestamp aproximado.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestID devuelve el ID de la request activa, o "" si el contexto no pasó por WithRequestID
// (ej. en tests).
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// Recover envuelve la cadena de middleware con recover(). Sin esto, un panic en cualquier
// handler dejaba la conexión rota (net/http recupera el panic para no tumbar el proceso, pero no
// deja ningún registro estructurado ni responde nada coherente al cliente). Va lo más afuera
// posible en la cadena para cubrir también cors/tenant, no solo los handlers de negocio.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recuperado",
						"request_id", RequestID(r.Context()),
						"method", r.Method,
						"path", r.URL.Path,
						"panic", rec,
						"stack", string(debug.Stack()),
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"error interno del servidor"}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
