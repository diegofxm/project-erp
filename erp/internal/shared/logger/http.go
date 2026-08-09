package logger

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// responseWriter envuelve http.ResponseWriter para capturar el status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

// Middleware registra cada request HTTP con el formato columnar.
//
// Formato de salida:
//
//	2026-07-27 20:15:33 | INFO  | http       | GET  /api/v1/sales    | 200 | 12ms
func Middleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: 0}

			next.ServeHTTP(rw, r)

			status := rw.status
			if status == 0 {
				status = http.StatusOK
			}
			dur := time.Since(start)
			msg := fmt.Sprintf("%-4s %s", r.Method, r.URL.Path)

			reqID := RequestID(r.Context())
			switch {
			case status >= 500:
				log.Error(msg, slog.Int("status", status), slog.Duration("dur", dur), slog.String("request_id", reqID))
			case status >= 400:
				log.Warn(msg, slog.Int("status", status), slog.Duration("dur", dur), slog.String("request_id", reqID))
			default:
				log.Info(msg, slog.Int("status", status), slog.Duration("dur", dur), slog.String("request_id", reqID))
			}
		})
	}
}
