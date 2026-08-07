// formatDateOnly formatea una fecha de calendario (sin componente horario significativo — ej.
// fecha de emisión, fecha de vencimiento, período de una declaración, TRM del día) sin pasar por
// `new Date(str).toLocaleDateString(...)`.
//
// El problema real: un string "2026-08-04" (o un instante ISO a medianoche UTC, que es como el
// backend Go serializa las fechas-sin-hora) lo interpreta JavaScript como medianoche UTC. Al
// mostrarlo con toLocaleDateString(), el navegador lo convierte a su zona horaria local — en
// Colombia (UTC-5) eso corre la fecha un día hacia atrás (2026-08-04 se ve como 3/8/2026).
//
// La corrección: extraer año/mes/día del string a mano — sin `options`, se arman directamente
// como DD/MM/AAAA (convención colombiana) sin pasar por Date/toLocaleDateString en absoluto, así
// no hay ninguna conversión de zona horaria de por medio que pueda correr la fecha. Si el llamador
// sí pasa `options` (ej. quiere el nombre del mes en letras), ahí sí se construye el Date con el
// constructor (year, month, day) — que interpreta esos componentes en hora LOCAL, sin conversión.
//
// Solo usar para fechas de calendario reales (issue_date, due_date, rate_date, period_start...).
// Para timestamps con hora significativa (created_at, closed_at, paid_at, filed_at...) la
// conversión de zona horaria de new Date(...).toLocaleDateString(...) sí es correcta — no
// reemplazar esos usos.
// bogotaTodayParts lee año/mes/día de HOY en el huso de Colombia (America/Bogota) — nunca usar
// `new Date().toISOString().slice(0, 10)` para "hoy": ese patrón lee el reloj UTC, que ya pasó a
// "mañana" para cualquiera en Colombia (UTC-5) desde las 7:00 p.m. en adelante. Es la otra cara
// del mismo problema de formatDateOnly de arriba, pero al calcular "hoy" en vez de mostrar una
// fecha ya guardada.
function bogotaTodayParts(): { y: number; m: number; d: number } {
  const parts = new Intl.DateTimeFormat("en-CA", { timeZone: "America/Bogota" }).formatToParts(new Date());
  const get = (type: string) => Number(parts.find((p) => p.type === type)?.value ?? 0);
  return { y: get("year"), m: get("month"), d: get("day") };
}

// todayColombiaISO devuelve la fecha de HOY en Colombia como YYYY-MM-DD — usar en vez de
// `new Date().toISOString().slice(0, 10)` para cualquier valor por defecto de un campo de fecha
// (fecha de emisión, "hoy" en un filtro, etc.).
export function todayColombiaISO(): string {
  const { y, m, d } = bogotaTodayParts();
  return `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

// addDaysColombiaISO devuelve la fecha a `days` días de HOY en Colombia (negativo = hacia atrás)
// como YYYY-MM-DD — para filtros tipo "últimos N días" o "vence en N días", con el mismo punto de
// partida correcto que todayColombiaISO.
export function addDaysColombiaISO(days: number): string {
  const { y, m, d } = bogotaTodayParts();
  const base = new Date(Date.UTC(y, m - 1, d));
  base.setUTCDate(base.getUTCDate() + days);
  return base.toISOString().slice(0, 10);
}

export function formatDateOnly(
  value: string | null | undefined,
  options?: Intl.DateTimeFormatOptions,
  locale = "es-CO",
): string {
  if (!value) return "—";
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value);
  if (!match) return value;
  const [, y, m, d] = match;
  if (!options) {
    return `${d}/${m}/${y}`;
  }
  const local = new Date(Number(y), Number(m) - 1, Number(d));
  return local.toLocaleDateString(locale, options);
}
