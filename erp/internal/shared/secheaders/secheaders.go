// Package secheaders agrega cabeceras de seguridad HTTP básicas a toda respuesta -- antes no
// existía ninguna (ver auditoría 2026-08-09, hallazgo "sin cabeceras de seguridad HTTP").
package secheaders

import "net/http"

// Middleware agrega X-Content-Type-Options, X-Frame-Options y una Content-Security-Policy
// mínima a toda respuesta. Este backend es una API JSON pura (no sirve HTML de la aplicación:
// el frontend es una SPA servida aparte) así que la CSP puede ser estricta -- no hay páginas
// propias que necesiten cargar scripts/estilos/imágenes.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// Evita que el navegador adivine un content-type distinto al declarado (mitiga XSS
		// vía archivos subidos que se sirven de vuelta, ej. PDFs/certificados).
		h.Set("X-Content-Type-Options", "nosniff")
		// Esta API nunca debe embeberse en un iframe -- mitiga clickjacking.
		h.Set("X-Frame-Options", "DENY")
		// default-src 'none' porque esta API no sirve HTML/JS/CSS propios; frame-ancestors
		// 'none' refuerza X-Frame-Options para navegadores que priorizan CSP sobre el header
		// legado.
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
