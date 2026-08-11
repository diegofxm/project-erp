-- reset_token/reset_token_expires_at: recuperación de contraseña por correo (forgot-password) --
-- mismo patrón que invite_token/invite_token_expires_at, columnas separadas para no mezclar los
-- dos flujos (invitar usuario nuevo vs. resetear la contraseña de uno que ya existe).
--
-- Migración NUEVA (no plegada en 000001) a propósito: production ya tiene datos reales y ya
-- corrió 000001, así que editar ese archivo nunca le habría llegado (golang-migrate no vuelve a
-- aplicar una versión ya registrada) -- justo el problema que causó el incidente de login del
-- 2026-08-11 con la columna token_version.
ALTER TABLE security.users
    ADD COLUMN reset_token UUID UNIQUE,
    ADD COLUMN reset_token_expires_at TIMESTAMPTZ;
