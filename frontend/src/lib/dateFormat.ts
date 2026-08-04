// formatDateOnly formatea una fecha de calendario (sin componente horario significativo — ej.
// fecha de emisión, fecha de vencimiento, período de una declaración, TRM del día) sin pasar por
// `new Date(str).toLocaleDateString(...)`.
//
// El problema real: un string "2026-08-04" (o un instante ISO a medianoche UTC, que es como el
// backend Go serializa las fechas-sin-hora) lo interpreta JavaScript como medianoche UTC. Al
// mostrarlo con toLocaleDateString(), el navegador lo convierte a su zona horaria local — en
// Colombia (UTC-5) eso corre la fecha un día hacia atrás (2026-08-04 se ve como 3/8/2026).
//
// La corrección: extraer año/mes/día del string a mano y construir el Date con el constructor
// (year, month, day), que interpreta esos componentes en hora LOCAL — sin conversión de por
// medio, no hay nada que corra.
//
// Solo usar para fechas de calendario reales (issue_date, due_date, rate_date, period_start...).
// Para timestamps con hora significativa (created_at, closed_at, paid_at, filed_at...) la
// conversión de zona horaria de new Date(...).toLocaleDateString(...) sí es correcta — no
// reemplazar esos usos.
export function formatDateOnly(
  value: string | null | undefined,
  options?: Intl.DateTimeFormatOptions,
  locale = "es-CO",
): string {
  if (!value) return "—";
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value);
  if (!match) return value;
  const [, y, m, d] = match;
  const local = new Date(Number(y), Number(m) - 1, Number(d));
  return options ? local.toLocaleDateString(locale, options) : local.toLocaleDateString(locale);
}
