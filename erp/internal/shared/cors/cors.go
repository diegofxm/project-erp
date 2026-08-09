package cors

import "net/http"

// Middleware agrega cabeceras CORS a todas las respuestas y responde 204 a preflights OPTIONS.
//
// allowedOrigins es la lista resuelta de orígenes permitidos (ver CORS_ALLOWED_ORIGINS en
// cmd/server/main.go, que exige la variable en vez de caer en modo permisivo si falta -- antes
// esta función leía os.Getenv directamente y trataba "vacío" igual que "*", quedando abierta a
// cualquier origen con credenciales por defecto si alguien olvidaba configurar la variable en
// producción). devMode, si es true, refleja cualquier origen -- debe ser una decisión explícita
// (CORS_DEV_MODE=true), nunca el resultado de omitir la configuración real.
func Middleware(allowedOrigins []string, devMode bool) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" && (devMode || allowed[origin]) {
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
}
