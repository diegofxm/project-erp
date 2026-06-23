package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

type Config struct {
	Env         Environment
	ServiceName string

	// Server
	Port int
	Host string

	// Database
	DatabaseURL     string
	DatabaseMaxConn int32
	DatabaseMinConn int32

	// Logging
	LogLevel string

	// IssuerSecretsKey cifra software_pin/certificate/certificate_password de internal/issuers
	// (AES-256-GCM) — nunca se guarda en texto plano en la base de datos. 32 bytes, base64.
	// Generar con: openssl rand -base64 32
	IssuerSecretsKey []byte

	// AuthJWTSecret firma los tokens de sesión de internal/auth (HS256) — deliberadamente
	// distinta de IssuerSecretsKey (cifra datos en reposo, no firma tokens; no deben
	// compartir clave). Mínimo 32 bytes, base64. Generar con: openssl rand -base64 32
	AuthJWTSecret []byte

	// CORSAllowedOrigins son los orígenes desde los que un navegador puede llamar a esta API
	// (otro puerto en desarrollo, el dominio del frontend en producción) — ver
	// internal/api/middleware/cors.go. Sin default a propósito: quién puede llamar a la API
	// desde un navegador es una decisión de seguridad explícita, no implícita. nil/vacío =
	// ningún navegador puede leer las respuestas (curl/Postman no se ven afectados). Un único
	// "*" permite cualquier origen — solo para desarrollo local, nunca producción.
	CORSAllowedOrigins []string
}

func Load() (*Config, error) {
	issuerSecretsKey, err := base64.StdEncoding.DecodeString(getEnv("ISSUER_SECRETS_KEY", ""))
	if err != nil {
		return nil, fmt.Errorf("ISSUER_SECRETS_KEY no es base64 válido: %w", err)
	}
	authJWTSecret, err := base64.StdEncoding.DecodeString(getEnv("AUTH_JWT_SECRET", ""))
	if err != nil {
		return nil, fmt.Errorf("AUTH_JWT_SECRET no es base64 válido: %w", err)
	}

	cfg := &Config{
		Env:                Environment(getEnv("APP_ENV", string(EnvDevelopment))),
		ServiceName:        getEnv("SERVICE_NAME", "apidian"),
		Port:               getEnvInt("PORT", 8080),
		Host:               getEnv("HOST", "0.0.0.0"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		DatabaseMaxConn:    int32(getEnvInt("DATABASE_MAX_CONN", 25)),
		DatabaseMinConn:    int32(getEnvInt("DATABASE_MIN_CONN", 5)),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		IssuerSecretsKey:   issuerSecretsKey,
		AuthJWTSecret:      authJWTSecret,
		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535, got %d", c.Port)
	}
	switch c.Env {
	case EnvDevelopment, EnvStaging, EnvProduction:
	default:
		return fmt.Errorf("APP_ENV must be development, staging or production, got %q", c.Env)
	}
	if len(c.IssuerSecretsKey) != 32 {
		return fmt.Errorf("ISSUER_SECRETS_KEY debe decodificar a 32 bytes (AES-256), tiene %d", len(c.IssuerSecretsKey))
	}
	if len(c.AuthJWTSecret) < 32 {
		return fmt.Errorf("AUTH_JWT_SECRET debe decodificar a al menos 32 bytes, tiene %d", len(c.AuthJWTSecret))
	}
	return nil
}

func (c *Config) IsProduction() bool {
	return c.Env == EnvProduction
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}

// getEnvList parsea una lista separada por comas (ej. CORS_ALLOWED_ORIGINS) — entradas vacías
// (espacios, comas dobles) se descartan. nil si la variable no está definida.
func getEnvList(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
