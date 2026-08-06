// Package seed siembra el primer superadmin de la plataforma, si está configurado.
package seed

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// All crea (o promueve) al superadmin inicial leyendo SUPERADMIN_EMAIL/SUPERADMIN_PASSWORD del
// entorno — nunca hardcodeado. Si no están definidas ambas, no hace nada (arrancar el servidor
// sin esas variables es válido, simplemente no hay superadmin todavía y hay que promoverlo a
// mano por SQL una vez, o definir las variables y reiniciar).
//
// Idempotente y seguro de correr en cada arranque: si el usuario ya existe, solo se asegura de
// que is_superadmin quede en TRUE — nunca pisa una contraseña que el usuario ya haya cambiado
// desde la app. Si no existe, se crea activo con la contraseña del entorno.
func All(ctx context.Context, pool *pgxpool.Pool) error {
	email := os.Getenv("SUPERADMIN_EMAIL")
	password := os.Getenv("SUPERADMIN_PASSWORD")
	if email == "" || password == "" {
		return nil
	}

	var existingID uuid.UUID
	err := pool.QueryRow(ctx, "SELECT id FROM security.users WHERE email = $1", email).Scan(&existingID)
	if err == nil {
		_, err := pool.Exec(ctx,
			"UPDATE security.users SET is_superadmin = TRUE, updated_at = NOW() WHERE id = $1",
			existingID,
		)
		if err != nil {
			return fmt.Errorf("seed superadmin: promover existente: %w", err)
		}
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("seed superadmin: hash de contraseña: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO security.users (id, email, password_hash, name, role, is_active, is_superadmin)
		VALUES ($1, $2, $3, $4, 'admin', TRUE, TRUE)
		ON CONFLICT (email) DO NOTHING`,
		uuid.New(), email, string(hash), "Superadmin",
	)
	if err != nil {
		return fmt.Errorf("seed superadmin: crear: %w", err)
	}
	return nil
}
