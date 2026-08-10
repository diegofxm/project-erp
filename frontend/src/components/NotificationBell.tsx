import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { AlertTriangle, Bell, BellOff, CheckCheck, X } from "lucide-react";
import { useNotifications } from "../context/NotificationContext";

export function NotificationBell() {
  const { notifications, unreadCount, markAsRead, markAllAsRead, dismiss } = useNotifications();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function onMouseDown(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", onMouseDown);
    return () => document.removeEventListener("mousedown", onMouseDown);
  }, [open]);

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        title="Notificaciones"
        aria-label={unreadCount > 0 ? `Notificaciones, ${unreadCount} sin leer` : "Notificaciones"}
        aria-haspopup="true"
        aria-expanded={open}
        className="relative rounded p-1.5 text-(--navbar-text) opacity-80 hover:bg-white/10 hover:opacity-100"
      >
        <Bell className="h-4 w-4" />
        {unreadCount > 0 && (
          <span className="absolute right-0.5 top-0.5 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-(--color-danger) px-0.5 text-[9px] font-bold leading-none text-white">
            {unreadCount > 9 ? "9+" : unreadCount}
          </span>
        )}
      </button>

      <div className={`absolute right-0 top-full z-50 mt-1 w-80 rounded border border-(--border-light) bg-(--bg-secondary) shadow-lg origin-top-right transition-all duration-150 ease-out ${open ? "opacity-100 scale-100" : "pointer-events-none opacity-0 scale-95"}`}>
          {/* Cabecera */}
          <div className="flex items-center justify-between border-b border-(--border-light) px-3 py-2">
            <span className="text-xs font-semibold text-(--text-primary)">Notificaciones</span>
            {unreadCount > 0 && (
              <button
                type="button"
                onClick={markAllAsRead}
                className="flex items-center gap-1 text-xs text-(--accent-primary) hover:underline"
              >
                <CheckCheck className="h-3 w-3" />
                Marcar todas leídas
              </button>
            )}
          </div>

          {/* Lista o estado vacío */}
          {notifications.length === 0 ? (
            <div className="flex flex-col items-center gap-2 px-4 py-8 text-center">
              <BellOff className="h-6 w-6 text-(--text-muted)" />
              <p className="text-xs text-(--text-muted)">Sin notificaciones activas</p>
            </div>
          ) : (
            <ul className="max-h-80 overflow-y-auto divide-y divide-(--border-light)">
              {notifications.map((n) => (
                <li key={n.id} className={`flex items-start gap-1 border-b border-(--border-light) last:border-b-0 hover:bg-(--bg-hover) transition-colors ${n.isRead ? "opacity-60" : ""}`}>
                  <Link
                    to={n.linkTo}
                    onClick={() => { markAsRead(n.id); setOpen(false); }}
                    className="flex min-w-0 flex-1 gap-2.5 px-3 py-2.5"
                  >
                    <AlertTriangle
                      className={`mt-0.5 h-3.5 w-3.5 shrink-0 ${
                        n.tone === "danger" ? "text-(--color-danger-text)" : "text-(--color-warning-text)"
                      }`}
                    />
                    <div className="min-w-0">
                      <p className={`text-xs text-(--text-primary) ${n.isRead ? "font-normal" : "font-semibold"}`}>
                        {n.title}
                      </p>
                      <p className="mt-0.5 text-xs text-(--text-secondary) leading-relaxed">{n.message}</p>
                    </div>
                  </Link>
                  <button
                    type="button"
                    title="Descartar"
                    aria-label={`Descartar notificación: ${n.title}`}
                    onClick={() => dismiss(n.id)}
                    className="mr-1 mt-2 shrink-0 rounded p-1 text-(--text-muted) hover:bg-(--bg-tertiary) hover:text-(--text-primary)"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
    </div>
  );
}
