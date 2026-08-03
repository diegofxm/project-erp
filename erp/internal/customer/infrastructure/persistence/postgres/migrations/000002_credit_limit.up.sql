-- Cupo de crédito del cliente — nil/NULL significa sin límite. Se valida al confirmar una
-- venta (sales/application/confirm.go) contra la cartera pendiente del cliente.
ALTER TABLE customer.customers ADD COLUMN IF NOT EXISTS credit_limit NUMERIC;
