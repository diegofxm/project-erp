-- token_version permite revocar sesiones activas del lado del servidor (logout real, invalidar
-- todo al cambiar contraseña) -- el JWT lleva su propio "tv" como claim; si no coincide con el
-- valor actual en esta columna, la sesión se considera inválida aunque la firma/expiración del
-- token sigan siendo válidas. Se incrementa en logout y en cambio de contraseña.
ALTER TABLE security.users ADD COLUMN token_version INT NOT NULL DEFAULT 0;
