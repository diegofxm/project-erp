package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

// loginBucket registra cuántas peticiones de login llegaron de una IP en la ventana actual.
type loginBucket struct {
	count int
	reset time.Time
}

var (
	loginMu      sync.Mutex
	loginBuckets = make(map[netip.Addr]*loginBucket)
	loginOnce    sync.Once
)

// LoginRateLimit aplica 10 peticiones por minuto por IP al endpoint de login — protección
// básica contra fuerza bruta sin dependencias externas (ventana fija de 1 minuto).
// La goroutine de limpieza se lanza una sola vez (sync.Once) para evitar acumular entradas
// de IPs que dejaron de intentar, sin que crezca el mapa indefinidamente.
func LoginRateLimit(next http.Handler) http.Handler {
	loginOnce.Do(func() {
		go func() {
			for range time.Tick(5 * time.Minute) {
				loginMu.Lock()
				cutoff := time.Now().Add(-2 * time.Minute)
				for ip, b := range loginBuckets {
					if b.reset.Before(cutoff) {
						delete(loginBuckets, ip)
					}
				}
				loginMu.Unlock()
			}
		}()
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, err := extractClientIP(r)
		if err != nil {
			// Si no se puede parsear la IP, se deja pasar — no bloquear por fallo interno.
			next.ServeHTTP(w, r)
			return
		}

		loginMu.Lock()
		b, ok := loginBuckets[ip]
		if !ok || time.Now().After(b.reset) {
			b = &loginBucket{reset: time.Now().Add(time.Minute)}
			loginBuckets[ip] = b
		}
		b.count++
		allowed := b.count <= 10
		loginMu.Unlock()

		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "demasiados intentos de inicio de sesión — espera un minuto y vuelve a intentarlo",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extractClientIP obtiene la IP del cliente respetando X-Forwarded-For (primer elemento de la
// cadena) — útil detrás de un proxy/load balancer donde RemoteAddr es la IP del proxy.
func extractClientIP(r *http.Request) (netip.Addr, error) {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			xff = xff[:i]
		}
		if addr, err := netip.ParseAddr(strings.TrimSpace(xff)); err == nil {
			return addr, nil
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return netip.ParseAddr(host)
}
