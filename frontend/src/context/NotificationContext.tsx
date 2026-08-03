import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useAuth } from "./AuthContext";
import { listSystemAlerts, type SystemAlert } from "../lib/systemAlerts";

const READS_KEY = "apidian.notification_reads";
const DISMISSED_KEY = "apidian.notification_dismissed";

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

// El cálculo (umbral de 30 días, textos, IDs determinísticos) vive en el backend —
// erp/internal/shared/notification. Acá solo se combina con leído/descartado, que
// sigue siendo estado local (no hay necesidad de sincronizarlo entre dispositivos).
function toNotifications(alerts: SystemAlert[], reads: Set<string>, dismissed: Set<string>): AppNotification[] {
  return alerts
    .filter((a) => !dismissed.has(a.id))
    .map((a) => ({
      id: a.id,
      tone: a.severity,
      title: a.title,
      message: a.message,
      isRead: reads.has(a.id),
      linkTo: a.link_to,
    }));
}

export function NotificationProvider({ children }: { children: ReactNode }) {
  const { activeCompany, isAuthenticated } = useAuth();
  const [alerts, setAlerts] = useState<SystemAlert[]>([]);
  const [reads, setReads] = useState<Set<string>>(() => readSet(READS_KEY));
  const [dismissed, setDismissed] = useState<Set<string>>(() => readSet(DISMISSED_KEY));

  useEffect(() => {
    if (!isAuthenticated || !activeCompany) { setAlerts([]); return; }
    listSystemAlerts().then(setAlerts).catch(() => setAlerts([]));
  }, [isAuthenticated, activeCompany?.id]);

  const notifications = useMemo(
    () => toNotifications(alerts, reads, dismissed),
    [alerts, reads, dismissed],
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
