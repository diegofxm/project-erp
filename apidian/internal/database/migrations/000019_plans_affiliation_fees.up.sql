-- Planes: tarifa de afiliación inicial, tarifa de renovación anual y tasa de incremento anual.
ALTER TABLE plans
    ADD COLUMN affiliation_fee_cop  INT            NOT NULL DEFAULT 180000,
    ADD COLUMN renewal_fee_cop      INT            NOT NULL DEFAULT 180000,
    ADD COLUMN annual_increment_pct NUMERIC(5,2)   NOT NULL DEFAULT 0;

-- Emisor: tracking de su afiliación — tarifas pactadas y fechas de vigencia.
ALTER TABLE issuer_settings
    ADD COLUMN affiliation_fee_cop INT         NOT NULL DEFAULT 180000,
    ADD COLUMN renewal_fee_cop     INT         NOT NULL DEFAULT 180000,
    ADD COLUMN affiliated_at       TIMESTAMPTZ,
    ADD COLUMN renewal_due_at      TIMESTAMPTZ;
