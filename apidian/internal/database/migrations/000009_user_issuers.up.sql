-- Fase 9.32: user_issuers — relación N:M entre usuarios y emisores. Reemplaza el modelo
-- rígido "un usuario = un emisor" (issuer_id fijo en users, ver 000007): un usuario se registra
-- SIN empresa (POST /auth/register ya no la pide) y crea/se vincula a una o varias después
-- (POST /issuers, multi-empresa/sucursales) — ver docs/apidian-architecture.md sección 9.32.
--
-- role no se hace cumplir todavía (cualquier fila aquí da acceso completo al emisor, igual
-- que el modelo anterior) — queda guardado para cuando haga falta un permiso más granular
-- (ej. "viewer" de solo lectura), sin necesitar otra migración para agregarlo.
--
-- ON DELETE CASCADE en ambos lados: si se borra un usuario o un emisor, sus vínculos
-- desaparecen con ellos — no debe quedar un user_issuers huérfano apuntando a ninguno de los
-- dos. (Hoy no existe DELETE de usuarios ni de emisores en la API; esto solo protege contra
-- un futuro DELETE que sí se agregue.)
CREATE TABLE user_issuers (
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    issuer_id  UUID         NOT NULL REFERENCES issuers(id) ON DELETE CASCADE,
    role       VARCHAR(20)  NOT NULL DEFAULT 'owner',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, issuer_id)
);

CREATE INDEX idx_user_issuers_issuer ON user_issuers(issuer_id);
