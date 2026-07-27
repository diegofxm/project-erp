package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/diegofxm/accounting"
	"github.com/diegofxm/apidian/internal/api"
	"github.com/diegofxm/apidian/internal/config"
	"github.com/diegofxm/apidian/internal/database"
	"github.com/diegofxm/apidian/internal/email"
)

// Server is the HTTP entry point for the apidian service.
type Server struct {
	cfg            *config.Config
	log            *zap.Logger
	db             *database.DB
	accountingCore *accounting.Core
	httpServer     *http.Server
}

// Options holds all dependencies required to build a Server.
type Options struct {
	Config         *config.Config
	Logger         *zap.Logger
	DB             *database.DB
	AccountingCore *accounting.Core
}

// New constructs a Server and wires the HTTP routes.
func New(opts Options) *Server {
	s := &Server{
		cfg:            opts.Config,
		log:            opts.Logger,
		db:             opts.DB,
		accountingCore: opts.AccountingCore,
	}

	s.httpServer = &http.Server{
		Addr:         opts.Config.Addr(),
		Handler:      s.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s
}

// routes registers all HTTP endpoints and returns the mux.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleHealth)

	// API de negocio (/api/v1/...) — solo si hay DB disponible, igual patrón que core-bank.
	if s.db != nil {
		smtpCfg := email.Config{
			Host:        s.cfg.SMTPHost,
			Port:        s.cfg.SMTPPort,
			Username:    s.cfg.SMTPUsername,
			Password:    s.cfg.SMTPPassword,
			FromAddress: s.cfg.SMTPFromAddress,
			FromName:    s.cfg.SMTPFromName,
		}
		mux.Handle("/api/", api.New(s.log, s.db, s.cfg.IssuerSecretsKey, s.cfg.AuthJWTSecret, s.cfg.CORSAllowedOrigins, smtpCfg, s.cfg.AppBaseURL, s.accountingCore).Handler())
	}

	return mux
}

// ServeHTTP allows using the server as an http.Handler in tests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.httpServer.Handler.ServeHTTP(w, r)
}

// Start begins listening and serving HTTP requests.
// It blocks until ctx is cancelled or a fatal error occurs.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.log.Info("server started", zap.String("addr", s.cfg.Addr()))
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		return s.Stop(context.Background())
	case err := <-errCh:
		return err
	}
}

// Stop gracefully drains active connections.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("server shutting down")

	shutdownCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	s.log.Info("server stopped")
	return nil
}
