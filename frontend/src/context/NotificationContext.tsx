import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useAuth } from "./AuthContext";
import { listNumberingRanges } from "../lib/numberingRanges";
import type { Issuer, NumberingRange } from "../lib/types";

const READS_KEY = "apidian.notification_reads";
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
}

const NotificationContext = createContext<NotificationContextValue | null>(null);

function readReads(): Set<string> {
  try {
    const raw = localStorage.getItem(READS_KEY);
    return raw ? new Set(JSON.parse(raw) as string[]) : new Set();
  } catch {
    return new Set();
  }
}

function saveReads(reads: Set<string>) {
  localStorage.setItem(READS_KEY, JSON.stringify([...reads]));
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

function computeNotifications(issuer: Issuer | null, ranges: NumberingRange[], reads: Set<string>): AppNotification[] {
  const out: AppNotification[] = [];

  // ── Certificado digital ────────────────────────────────────────────────────
  if (issuer?.has_certificate && issuer.certificate_expires_at) {
    const days = daysUntil(issuer.certificate_expires_at);
    const exp = fmtDate(issuer.certificate_expires_at);

    if (days < 0) {
      const id = `cert_expired:${issuer.certificate_expires_at}`;
      out.push({
        id, tone: "danger",
        title: "Certificado digital vencido",
        message: `Venció el ${exp}. No puedes firmar ni enviar documentos a la DIAN hasta renovarlo.`,
        isRead: reads.has(id),
        linkTo: "/settings/company",
      });
    } else if (days <= WARN_DAYS) {
      const id = `cert_expiring:${issuer.certificate_expires_at}`;
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
    }
  }

  return out;
}

export function NotificationProvider({ children }: { children: ReactNode }) {
  const { activeIssuer, isAuthenticated } = useAuth();
  const [ranges, setRanges] = useState<NumberingRange[]>([]);
  const [reads, setReads] = useState<Set<string>>(readReads);

  useEffect(() => {
    if (!isAuthenticated || !activeIssuer) { setRanges([]); return; }
    listNumberingRanges().then(setRanges).catch(() => setRanges([]));
  }, [isAuthenticated, activeIssuer?.id]);

  const notifications = useMemo(
    () => computeNotifications(activeIssuer, ranges, reads),
    [activeIssuer, ranges, reads],
  );

  const unreadCount = useMemo(() => notifications.filter((n) => !n.isRead).length, [notifications]);

  const markAsRead = useCallback((id: string) => {
    setReads((prev) => {
      const next = new Set(prev);
      next.add(id);
      saveReads(next);
      return next;
    });
  }, []);

  const markAllAsRead = useCallback(() => {
    setReads((prev) => {
      const next = new Set(prev);
      notifications.forEach((n) => next.add(n.id));
      saveReads(next);
      return next;
    });
  }, [notifications]);

  const value = useMemo<NotificationContextValue>(
    () => ({ notifications, unreadCount, markAsRead, markAllAsRead }),
    [notifications, unreadCount, markAsRead, markAllAsRead],
  );

  return <NotificationContext.Provider value={value}>{children}</NotificationContext.Provider>;
}

export function useNotifications(): NotificationContextValue {
  const ctx = useContext(NotificationContext);
  if (!ctx) throw new Error("useNotifications debe usarse dentro de <NotificationProvider>");
  return ctx;
}
