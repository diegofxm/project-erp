package cors

import (
	"net/http"
	"os"
	"strings"
)

// Middleware agrega cabeceras CORS a todas las respuestas y responde 204 a preflights OPTIONS.
// Orígenes permitidos se leen de la variable CORS_ALLOWED_ORIGINS (coma-separados).
// Si la variable no está definida, permite cualquier origen en desarrollo.
func Middleware(next http.Handler) http.Handler {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	allowed := map[string]bool{}
	wildcard := false
	if raw == "" || raw == "*" {
		wildcard = true
	} else {
		for _, o := range strings.Split(raw, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowed[o] = true
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" && (wildcard || allowed[origin]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
