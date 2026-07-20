package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/diegofxm/apidian/internal/config"
	"github.com/diegofxm/apidian/internal/database"
	"github.com/diegofxm/apidian/internal/logger"
	"github.com/diegofxm/apidian/internal/pdf"
	"github.com/diegofxm/apidian/internal/server"
)

const version = "0.1.0"

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Carga .env si existe (ignorado en producción donde las vars vienen del entorno)
	_ = godotenv.Load()

	// 1. Configuración
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 2. PDF: ruta a Chrome/Chromium (configurable vía CHROME_PATH; vacío = auto-detect en PATH)
	pdf.SetChromePath(cfg.ChromePath)

	// 3. Logger
	log, err := logger.New(cfg.LogLevel, cfg.IsProduction())
	if err != nil {
		return err
	}
	defer log.Sync() //nolint:errcheck

	log.Info("starting apidian",
		zap.String("version", version),
		zap.String("env", string(cfg.Env)),
		zap.String("addr", cfg.Addr()),
	)

	// 4. Base de datos
	db, err := database.New(ctx, database.Config{
		URL:     cfg.DatabaseURL,
		MaxConn: cfg.DatabaseMaxConn,
		MinConn: cfg.DatabaseMinConn,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	log.Info("database connected")

	if err := db.Migrate(); err != nil {
		return err
	}

	log.Info("migrations applied")

	if err := db.Seed(ctx); err != nil {
		return err
	}

	log.Info("catalogs seeded")

	if err := db.SeedPlans(ctx); err != nil {
		return err
	}

	log.Info("plans seeded")

	// 5. Servidor HTTP
	srv := server.New(server.Options{
		Config: cfg,
		Logger: log,
		DB:     db,
	})

	if err := srv.Start(ctx); err != nil {
		return err
	}

	return nil
}
