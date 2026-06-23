package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// Recovery catches panics and returns 500 instead of crashing the server.
func Recovery(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						zap.String("request_id", GetRequestID(r.Context())),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("panic", fmt.Sprintf("%v", rec)),
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "error interno del servidor"})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
