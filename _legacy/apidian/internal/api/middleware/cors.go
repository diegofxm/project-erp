package middleware

import "net/http"

// allowedMethods y allowedHeaders son fijos porque describen exactamente lo que esta API
// expone — GET/POST/PUT/DELETE en las rutas, y Content-Type/Authorization son los únicos dos
// headers no triviales que cualquier cliente de esta API necesita mandar.
const (
	allowedMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowedHeaders = "Content-Type, Authorization"
)

// CORS habilita que un frontend en el navegador, corriendo en otro origen (otro puerto en
// desarrollo, el dominio real en producción), pueda llamar a esta API. Sin esto, el navegador
// bloquea la respuesta aunque la petición sí le llegue al servidor — curl/Postman/llamadas
// servidor-a-servidor no se ven afectados, CORS solo lo aplica el navegador.
//
// Como casi toda esta API usa POST/PUT/DELETE, Content-Type: application/json, y exige el
// header no-estándar "Authorization: Bearer <token>", el navegador manda primero un
// "preflight" OPTIONS antes de la petición real — este middleware tiene que responderlo
// directamente (antes de llegar al mux, que no tiene rutas OPTIONS registradas), no solo
// decorar la respuesta final.
//
// allowedOrigins es la lista explícita de orígenes permitidos — sin default a propósito,
// mismo criterio que ISSUER_SECRETS_KEY/AUTH_JWT_SECRET (config.Config): quién puede llamar a
// la API desde un navegador es una decisión de seguridad explícita, nunca implícita. Un único
// "*" permite cualquier origen — pensado solo para desarrollo local, nunca producción.
//
// Esta API se autentica con "Authorization: Bearer <token>", no con cookies — por eso nunca
// manda Access-Control-Allow-Credentials, y por eso "*" es seguro de ofrecer cuando se pide
// explícitamente: no hay sesión de cookie que un sitio distinto pueda reusar a través de ahí.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			switch {
			case allowAll:
				w.Header().Set("Access-Control-Allow-Origin", "*")
			case origin != "" && allowed[origin]:
				// Nunca un literal "*" cuando la lista es explícita — hay que reflejar
				// exactamente el origen que vino, y avisar que la respuesta varía según ese
				// header (para que cachés intermedios no mezclen respuestas de orígenes
				// distintos).
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			default:
				// Origen ausente (curl/Postman/servidor-a-servidor) o no permitido: se deja
				// pasar la petición igual — CORS no es una puerta de autorización del
				// servidor, es información para que el navegador decida si le muestra la
				// respuesta a su propio JS. Sin estos headers, el navegador la bloquea solo.
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			w.Header().Set("Access-Control-Max-Age", "600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
