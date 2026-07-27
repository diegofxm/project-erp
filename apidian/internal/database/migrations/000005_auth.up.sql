-- Autenticación y acceso: relación usuario↔empresa + invitaciones.
-- Un usuario puede no tener empresa todavía (registro desacoplado)
-- o tener varias (multi-empresa/sucursales).

CREATE TABLE user_issuers (
    user_id    UUID        NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    issuer_id  UUID        NOT NULL REFERENCES issuers(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL DEFAULT 'owner',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, issuer_id)
);

CREATE INDEX idx_user_issuers_issuer ON user_issuers(issuer_id);
