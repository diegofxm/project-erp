package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	catalogpostgres "github.com/diegofxm/erp/internal/catalog/infrastructure/persistence/postgres"
	"github.com/diegofxm/erp/internal/catalog/infrastructure/persistence/postgres/seed"
	cataloghttp "github.com/diegofxm/erp/internal/catalog/interfaces/http"
	"github.com/diegofxm/erp/internal/shared/events"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

func main() {
	// Carga erp/.env si existe; en producción las variables vienen del entorno directamente.
	_ = godotenv.Load()

	databaseURL := mustEnv("DATABASE_URL")
	addr := envOr("ADDR", ":8080")

	// ── Base de datos ──────────────────────────────────────────────────────────
	pool := mustOpenDB(databaseURL)
	defer pool.Close()

	// ── Migraciones ─────────────────────────────────────────────────────────────
	mustMigrate("catalog", catalogpostgres.Migrate(databaseURL))
	// mustMigrate("company", companypostgres.Migrate(databaseURL))
	// mustMigrate("security", securitypostgres.Migrate(databaseURL))
	// mustMigrate("customer", customerpostgres.Migrate(databaseURL))

	// ── Bus de eventos ──────────────────────────────────────────────────────────
	bus := events.NewBus()
	// Suscripciones entre módulos — ejemplo:
	//   bus.Subscribe("invoice.confirmed", accounting.OnInvoiceConfirmed)
	//   bus.Subscribe("invoice.confirmed", inventory.OnInvoiceConfirmed)
	//   bus.Subscribe("invoice.confirmed", electronic.OnInvoiceConfirmed)
	//   bus.Subscribe("payroll.generated", accounting.OnPayrollGenerated)
	//   bus.Subscribe("payroll.generated", electronic.OnPayrollGenerated)
	_ = bus

	// ── Seed ────────────────────────────────────────────────────────────────────
	if err := seed.All(context.Background(), pool); err != nil {
		log.Fatalf("seed catálogos: %v", err)
	}

	// ── Repositorios ────────────────────────────────────────────────────────────
	catalogRepo := catalogpostgres.NewRepository(pool)
	// customerRepo := customerpostgres.NewRepository(pool)

	// ── Casos de uso ────────────────────────────────────────────────────────────
	// createCustomer := customerapp.NewCreateCustomer(customerRepo, bus)

	// ── Handlers HTTP ────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	cataloghttp.NewHandler(catalogRepo).RegisterRoutes(mux)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// ── Middleware ───────────────────────────────────────────────────────────────
	// El middleware de tenant siempre envuelve el mux completo — extrae company_id
	// del JWT y lo inyecta al contexto antes de que cualquier handler corra.
	handler := tenant.Middleware(mux)

	log.Printf("servidor ERP en %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("variable de entorno requerida: %s", key)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustOpenDB(url string) *pgxpool.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatalf("abrir base de datos: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping base de datos: %v", err)
	}
	return pool
}

func mustMigrate(module string, err error) {
	if err != nil {
		log.Fatalf("migración %s: %v", module, err)
	}
}
