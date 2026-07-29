import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useAuth } from "./AuthContext";
import { listNumberingRanges } from "../lib/numberingRanges";
import type { Company, NumberingRange } from "../lib/types";

const READS_KEY = "apidian.notification_reads";
const DISMISSED_KEY = "apidian.notification_dismissed";
const WARN_DAYS = 30;

export interface AppNotification {
  id: string;
  tone: "warning" | "danger";
  title: string;
  message: string;
  isRead: boolean;
  linkTo: string;
}

interface NotificationContextValue {
  notifications: AppNotification[];
  unreadCount: number;
  markAsRead: (id: string) => void;
  markAllAsRead: () => void;
  // dismiss descarta permanentemente una notificación — no reaparece hasta que cambie el
  // dato subyacente (ej. nueva fecha de vencimiento = nuevo ID = nueva notificación).
  dismiss: (id: string) => void;
}

const NotificationContext = createContext<NotificationContextValue | null>(null);

function readSet(key: string): Set<string> {
  try {
    const raw = localStorage.getItem(key);
    return raw ? new Set(JSON.parse(raw) as string[]) : new Set();
  } catch {
    return new Set();
  }
}

function saveSet(key: string, set: Set<string>) {
  localStorage.setItem(key, JSON.stringify([...set]));
}

function daysUntil(dateStr: string): number {
  return Math.floor((new Date(dateStr).getTime() - Date.now()) / 86_400_000);
}

function fmtDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString("es-CO", { day: "numeric", month: "short", year: "numeric" });
}

function docTypeLabel(code: string): string {
  return code === "01" ? "Factura" : code === "91" ? "Nota Crédito" : code === "92" ? "Nota Débito" : code;
}

function computeNotifications(company: Company | null, ranges: NumberingRange[], reads: Set<string>, dismissed: Set<string>): AppNotification[] {
  const out: AppNotification[] = [];

  // ── Certificado digital ────────────────────────────────────────────────────
  if (company?.has_certificate && company.certificate_expires_at) {
    const days = daysUntil(company.certificate_expires_at);
    const exp = fmtDate(company.certificate_expires_at);

    if (days < 0) {
      const id = `cert_expired:${company.certificate_expires_at}`;
      out.push({
        id, tone: "danger",
        title: "Certificado digital vencido",
        message: `Venció el ${exp}. No puedes firmar ni enviar documentos a la DIAN hasta renovarlo.`,
        isRead: reads.has(id),
        linkTo: "/settings/company",
      });
    } else if (days <= WARN_DAYS) {
      const id = `cert_expiring:${company.certificate_expires_at}`;
      out.push({
        id, tone: "warning",
        title: `Certificado digital vence en ${days} día${days === 1 ? "" : "s"}`,
        message: `Vence el ${exp}. Renuévalo antes para no interrumpir la facturación.`,
        isRead: reads.has(id),
        linkTo: "/settings/company",
      });
    }
  }

  // ── Rangos de numeración ───────────────────────────────────────────────────
  for (const r of ranges) {
    const rangeLabel = r.prefix ? `"${r.prefix}" (${docTypeLabel(r.dian_document_type_code)})` : docTypeLabel(r.dian_document_type_code);

    if (r.status === "expired") {
      const id = `range_expired:${r.id}`;
      out.push({
        id, tone: "danger",
        title: `Resolución ${rangeLabel} vencida`,
        message: `Venció el ${fmtDate(r.valid_to)}. Sin resolución vigente no puedes numerar documentos nuevos.`,
        isRead: reads.has(id),
        linkTo: "/settings/company",
      });
    } else if (r.status === "active") {
      const days = daysUntil(r.valid_to);
      if (days >= 0 && days <= WARN_DAYS) {
        const id = `range_expiring:${r.id}:${r.valid_to}`;
        out.push({
          id, tone: "warning",
          title: `Resolución ${rangeLabel} vence en ${days} día${days === 1 ? "" : "s"}`,
          message: `Vigente hasta el ${fmtDate(r.valid_to)}. Tramita la renovación ante la DIAN con anticipación.`,
          isRead: reads.has(id),
          linkTo: "/settings/company",
        });
      }
    } else if (r.status === "exhausted") {
      const id = `range_exhausted:${r.id}`;
      out.push({
        id, tone: "danger",
        title: `Resolución ${rangeLabel} agotada`,
        message: `Se usaron todos los números autorizados. Registra una nueva resolución para seguir facturando.`,
        isRead: reads.has(id),
        linkTo: "/settings/company",
      });
    }
  }

  // Las notificaciones descartadas no aparecen hasta que cambie el dato subyacente
  // (nuevo ID = nueva condición = reaparece sin leer).
  return out.filter((n) => !dismissed.has(n.id));
}

export function NotificationProvider({ children }: { children: ReactNode }) {
  const { activeCompany, isAuthenticated } = useAuth();
  const [ranges, setRanges] = useState<NumberingRange[]>([]);
  const [reads, setReads] = useState<Set<string>>(() => readSet(READS_KEY));
  const [dismissed, setDismissed] = useState<Set<string>>(() => readSet(DISMISSED_KEY));

  useEffect(() => {
    if (!isAuthenticated || !activeCompany) { setRanges([]); return; }
    listNumberingRanges().then(setRanges).catch(() => setRanges([]));
  }, [isAuthenticated, activeCompany?.id]);

  const notifications = useMemo(
    () => computeNotifications(activeCompany, ranges, reads, dismissed),
    [activeCompany, ranges, reads, dismissed],
  );

  const unreadCount = useMemo(() => notifications.filter((n) => !n.isRead).length, [notifications]);

  const markAsRead = useCallback((id: string) => {
    setReads((prev) => {
      const next = new Set(prev);
      next.add(id);
      saveSet(READS_KEY, next);
      return next;
    });
  }, []);

  const markAllAsRead = useCallback(() => {
    setReads((prev) => {
      const next = new Set(prev);
      notifications.forEach((n) => next.add(n.id));
      saveSet(READS_KEY, next);
      return next;
    });
  }, [notifications]);

  const dismiss = useCallback((id: string) => {
    // Descartar también implica leer — así no queda contando en el badge si se deshace.
    setDismissed((prev) => {
      const next = new Set(prev);
      next.add(id);
      saveSet(DISMISSED_KEY, next);
      return next;
    });
    setReads((prev) => {
      const next = new Set(prev);
      next.add(id);
      saveSet(READS_KEY, next);
      return next;
    });
  }, []);

  const value = useMemo<NotificationContextValue>(
    () => ({ notifications, unreadCount, markAsRead, markAllAsRead, dismiss }),
    [notifications, unreadCount, markAsRead, markAllAsRead, dismiss],
  );

  return <NotificationContext.Provider value={value}>{children}</NotificationContext.Provider>;
}

export function useNotifications(): NotificationContextValue {
  const ctx = useContext(NotificationContext);
  if (!ctx) throw new Error("useNotifications debe usarse dentro de <NotificationProvider>");
  return ctx;
}
