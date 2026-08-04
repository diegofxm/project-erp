-- Los defaults originales (424505/529005) no existen en el PUC real (extraído de puc.com.co) —
-- ver 424810 "Otros activos" y 531040 "Pérdidas por siniestros" como los reemplazos genéricos
-- más cercanos disponibles en el catálogo.
ALTER TABLE accounting.fixed_assets ALTER COLUMN gain_account SET DEFAULT '424810';
ALTER TABLE accounting.fixed_assets ALTER COLUMN loss_account SET DEFAULT '531040';
