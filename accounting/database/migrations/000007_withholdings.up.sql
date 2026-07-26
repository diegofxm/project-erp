-- Catálogo de conceptos de retención en la fuente, reteiva y reteica.
-- rate_bp: tarifa en basis points (1 bp = 0.01%; ej. 250 = 2.5%).
-- min_base_uvt: base mínima en UVT por debajo de la cual NO se practica retención.
--   0 significa que siempre aplica independientemente del monto.
CREATE TABLE IF NOT EXISTS accounting.withholding_concepts (
    id                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    code                VARCHAR(10)   NOT NULL,
    name                VARCHAR(200)  NOT NULL,
    type                VARCHAR(20)   NOT NULL CHECK (type IN ('RETEFUENTE','RETEIVA','RETEICA')),
    rate_bp             INTEGER       NOT NULL CHECK (rate_bp > 0),
    min_base_uvt        NUMERIC(10,2) NOT NULL DEFAULT 0,
    account_payable     VARCHAR(10)   NOT NULL,
    account_receivable  VARCHAR(10)   NOT NULL,
    applicable_to       VARCHAR(10)   NOT NULL DEFAULT 'BOTH'
                            CHECK (applicable_to IN ('NATURAL','JURIDICA','BOTH')),
    is_active           BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (code, type, applicable_to)
);

-- Valor del UVT (Unidad de Valor Tributario) por año, en centavos.
-- Publicado anualmente por la DIAN (Resolución de UVT).
CREATE TABLE IF NOT EXISTS accounting.uvt_values (
    year        INTEGER PRIMARY KEY,
    value_cents BIGINT  NOT NULL CHECK (value_cents > 0)
);

CREATE INDEX IF NOT EXISTS withholding_concepts_type_idx ON accounting.withholding_concepts (type);
CREATE INDEX IF NOT EXISTS withholding_concepts_active_idx ON accounting.withholding_concepts (is_active)
    WHERE is_active = TRUE;
