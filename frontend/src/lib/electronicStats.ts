// Métricas de facturación electrónica calculadas en el cliente a partir de la lista de
// documentos ya cargada (mismo patrón que sales/purchase/inventory/accounting: sin endpoint
// de backend dedicado a estadísticas, se agrega lo que el listado ya trae). Reemplaza el antiguo
// GET /stats/billing (módulo stats/, eliminado 2026-08-09 por quedar huérfano y por romper el
// mismo estándar que este archivo ahora sigue -- ver docs/auditorias/2026-08-09/01-arquitectura.md).
import type { Document, DocumentStatus } from "./types";

export interface PeriodStats {
  revenue_cents: number;
  document_count: number;
  accepted_count: number;
  rejected_count: number;
  draft_count: number;
}

export interface TypeStats {
  type_code: string;
  type_name: string;
  count: number;
  revenue_cents: number;
}

export interface MonthSeries {
  month: string; // "YYYY-MM"
  revenue_cents: number;
  count: number;
  accepted_count: number;
}

export interface BillingStats {
  current_month: PeriodStats;
  previous_month: PeriodStats;
  ytd: PeriodStats;
  by_type: TypeStats[];
  series: MonthSeries[];
}

const REJECTED_STATUSES: DocumentStatus[] = ["rejected", "send_error", "send_unknown", "environment_mismatch"];

const TYPE_NAMES: Record<string, string> = {
  "01": "Factura Electrónica",
  "91": "Nota Crédito",
  "92": "Nota Débito",
  "05": "Documento Soporte",
};

function monthKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
}

function emptyPeriod(): PeriodStats {
  return { revenue_cents: 0, document_count: 0, accepted_count: 0, rejected_count: 0, draft_count: 0 };
}

function inRange(date: Date, start: Date, end: Date): boolean {
  return date >= start && date < end;
}

// computeBillingStats replica los 3 agregados que antes hacía el backend por SQL
// (period stats, por tipo, serie 12 meses), ahora en memoria sobre los documentos que
// listDocuments() ya trajo -- exacto mismo criterio: revenue solo de status="accepted",
// rejected agrupa rejected/send_error/send_unknown/environment_mismatch.
export function computeBillingStats(documents: Document[], now: Date = new Date()): BillingStats {
  const startOfMonth = new Date(now.getFullYear(), now.getMonth(), 1);
  const startOfPrevMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1);
  const startOfYear = new Date(now.getFullYear(), 0, 1);
  const now_ = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1); // límite superior exclusivo

  const currentMonth = emptyPeriod();
  const previousMonth = emptyPeriod();
  const ytd = emptyPeriod();
  const byType = new Map<string, TypeStats>();
  const series = new Map<string, MonthSeries>();

  for (const doc of documents) {
    if (doc.status === "draft") currentMonth.draft_count += 1;

    if (!doc.issue_date) continue;
    const issueDate = new Date(doc.issue_date);
    const revenue = doc.totals?.payable_cents ?? 0;
    const isAccepted = doc.status === "accepted";
    const isRejected = REJECTED_STATUSES.includes(doc.status);
    const isNotDraft = doc.status !== "draft";

    if (inRange(issueDate, startOfMonth, now_)) {
      if (isAccepted) currentMonth.revenue_cents += revenue;
      if (isNotDraft) currentMonth.document_count += 1;
      if (isAccepted) currentMonth.accepted_count += 1;
      if (isRejected) currentMonth.rejected_count += 1;
    }
    if (inRange(issueDate, startOfPrevMonth, startOfMonth)) {
      if (isAccepted) previousMonth.revenue_cents += revenue;
      if (isNotDraft) previousMonth.document_count += 1;
      if (isAccepted) previousMonth.accepted_count += 1;
      if (isRejected) previousMonth.rejected_count += 1;
    }
    if (inRange(issueDate, startOfYear, now_)) {
      if (isAccepted) ytd.revenue_cents += revenue;
      if (isNotDraft) ytd.document_count += 1;
      if (isAccepted) ytd.accepted_count += 1;
    }

    if (inRange(issueDate, startOfMonth, now_) && isNotDraft) {
      const key = doc.dian_document_type_code;
      const entry = byType.get(key) ?? { type_code: key, type_name: TYPE_NAMES[key] ?? key, count: 0, revenue_cents: 0 };
      entry.count += 1;
      if (isAccepted) entry.revenue_cents += revenue;
      byType.set(key, entry);
    }

    const seriesStart = new Date(startOfMonth.getFullYear(), startOfMonth.getMonth() - 11, 1);
    if (isNotDraft && inRange(issueDate, seriesStart, now_)) {
      const key = monthKey(issueDate);
      const entry = series.get(key) ?? { month: key, revenue_cents: 0, count: 0, accepted_count: 0 };
      entry.count += 1;
      if (isAccepted) {
        entry.accepted_count += 1;
        entry.revenue_cents += revenue;
      }
      series.set(key, entry);
    }
  }

  return {
    current_month: currentMonth,
    previous_month: previousMonth,
    ytd,
    by_type: [...byType.values()].sort((a, b) => b.count - a.count),
    series: [...series.values()].sort((a, b) => a.month.localeCompare(b.month)),
  };
}
