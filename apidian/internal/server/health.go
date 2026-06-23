package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type healthStatus string

const (
	statusOK       healthStatus = "ok"
	statusDegraded healthStatus = "degraded"
)

type healthResponse struct {
	Status    healthStatus      `json:"status"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := make(map[string]string)
	overall := statusOK

	if s.db != nil {
		if err := s.db.Ping(ctx); err != nil {
			checks["database"] = "unhealthy: " + err.Error()
			overall = statusDegraded
		} else {
			checks["database"] = "ok"
		}
	} else {
		checks["database"] = "not configured"
	}

	resp := healthResponse{
		Status:    overall,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}

	code := http.StatusOK
	if overall == statusDegraded {
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}
