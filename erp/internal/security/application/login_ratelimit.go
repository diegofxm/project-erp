package application

import (
	"sync"
	"time"
)

// LoginRateLimiter bloquea intentos de login tras demasiados fallos recientes para el mismo
// correo -- antes no existía ningún límite, un atacante podía probar contraseñas contra una
// cuenta sin restricción (ver auditoría 2026-08-09, "sin rate limiting ni protección de fuerza
// bruta en login"). Implementación en memoria (no Redis/BD): a la escala real de este ERP un
// mapa protegido por mutex es suficiente, y evita sumar infraestructura nueva solo para esto.
// Limitación conocida: el contador vive en el proceso, así que se reinicia si el servidor se
// reinicia, y no se comparte entre réplicas si algún día el backend corre en más de un proceso.
type LoginRateLimiter struct {
	mu          sync.Mutex
	attempts    map[string]*loginAttempt
	maxAttempts int
	window      time.Duration
	lockout     time.Duration
}

type loginAttempt struct {
	count       int
	windowStart time.Time
	lockedUntil time.Time
}

// NewLoginRateLimiter usa 5 intentos fallidos en 15 minutos como umbral, con 15 minutos de
// bloqueo -- valores de sentido común, no derivados de un estudio de tráfico real; son fáciles
// de ajustar si en producción resultan demasiado o muy poco estrictos.
func NewLoginRateLimiter() *LoginRateLimiter {
	return &LoginRateLimiter{
		attempts:    make(map[string]*loginAttempt),
		maxAttempts: 5,
		window:      15 * time.Minute,
		lockout:     15 * time.Minute,
	}
}

// Allow indica si se puede intentar un login para key (normalmente el correo, en minúsculas).
func (l *LoginRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	a, ok := l.attempts[key]
	if !ok {
		return true
	}
	return time.Now().After(a.lockedUntil)
}

// RecordFailure cuenta un intento fallido; si llega al umbral dentro de la ventana, bloquea key
// por l.lockout. También aprovecha la llamada para barrer entradas vencidas de otras keys y que
// el mapa no crezca indefinidamente (ej. alguien probando muchos correos distintos).
func (l *LoginRateLimiter) RecordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()

	a, ok := l.attempts[key]
	if !ok || now.Sub(a.windowStart) > l.window {
		a = &loginAttempt{windowStart: now}
		l.attempts[key] = a
	}
	a.count++
	if a.count >= l.maxAttempts {
		a.lockedUntil = now.Add(l.lockout)
	}

	for k, v := range l.attempts {
		if k != key && now.Sub(v.windowStart) > l.window && now.After(v.lockedUntil) {
			delete(l.attempts, k)
		}
	}
}

// RecordSuccess limpia el contador de key -- un login correcto no debe dejar arrastrando
// intentos fallidos previos que ya no importan.
func (l *LoginRateLimiter) RecordSuccess(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}
