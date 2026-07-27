package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// RequireTenant exige que el token ya tenga una empresa activa (GetTenantID != uuid.Nil) —
// va DESPUÉS de Auth en la cadena. Desde la Fase 9.32, un usuario autenticado puede no tener
// ninguna empresa todavía (registro desacoplado) o tener varias sin haber elegido cuál usar
// — casi toda la API sigue necesitando una empresa activa para saber a quién pertenece cada
// recurso, así que se corta aquí con un mensaje claro en vez de dejar que cada handler
// silenciosamente devuelva listas vacías o falle de forma confusa más abajo.
//
// Las rutas de gestión de empresas (POST/GET /issuers, POST /issuers/{id}/select) NO usan
// este middleware a propósito — son justamente las que un usuario sin empresa activa
// necesita poder llamar.
func RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetTenantID(r.Context()) == uuid.Nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "no tienes una empresa activa — créala (POST /issuers) o selecciona una (POST /issuers/{id}/select)",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
