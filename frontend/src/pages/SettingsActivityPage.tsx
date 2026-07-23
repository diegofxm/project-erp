import { useEffect, useState } from "react";
import { Activity } from "lucide-react";
import { useNavigate } from "react-router";
import { listAuditEvents } from "../lib/audit";
import type { AuditEvent } from "../lib/types";
import { ApiError } from "../lib/apiClient";
import { Banner } from "../components/ui/Banner";
import { Spinner } from "../components/ui/Spinner";

const ACTION_LABELS: Record<string, string> = {
  "document.created":    "Borrador creado",
  "document.updated":    "Borrador actualizado",
  "document.confirmed":  "Confirmado",
  "document.deleted":    "Eliminado",
  "document.email_sent": "Correo enviado",
  "document.cloned":     "Clonado",
};

const ACTION_COLORS: Record<string, string> = {
  "document.created":    "text-(--color-info-text) bg-(--color-info-bg)",
  "document.updated":    "text-(--text-secondary) bg-(--bg-secondary)",
  "document.confirmed":  "text-(--color-success-text) bg-(--color-success-bg)",
  "document.deleted":    "text-(--color-danger-text) bg-(--color-danger-bg)",
  "document.email_sent": "text-(--accent-primary) bg-(--bg-secondary)",
  "document.cloned":     "text-(--text-secondary) bg-(--bg-secondary)",
};

const DOC_TYPE_LABELS: Record<string, string> = {
  "01": "Factura",
  "91": "Nota Crédito",
  "92": "Nota Débito",
  "05": "Doc. Soporte",
  "95": "Nota Ajuste",
};

const ENV_LABELS: Record<string, string> = {
  "1": "Producción",
  "2": "Habilitación",
};

function eventDocRef(meta?: Record<string, unknown>): string {
  if (!meta) return "—";
  const prefix = meta.prefix as string | undefined;
  const number = meta.number as number | undefined;
  if (prefix && number) return `${prefix}-${number}`;
  if (prefix) return prefix;
  return "—";
}

function eventDocType(meta?: Record<string, unknown>): string {
  if (!meta) return "";
  const typeCode = meta.dian_document_type_code as string | undefined;
  return typeCode ? (DOC_TYPE_LABELS[typeCode] ?? typeCode) : "";
}

function eventParty(meta?: Record<string, unknown>): string {
  if (!meta) return "—";
  return ((meta.customer_name ?? meta.vendor_name ?? "") as string) || "—";
}

function eventEnv(meta?: Record<string, unknown>): string {
  if (!meta) return "";
  const env = meta.environment as string | undefined;
  return env ? (ENV_LABELS[env] ?? "") : "";
}

const PAGE_SIZE = 20;

export function SettingsActivityPage() {
  const navigate = useNavigate();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(0);

  useEffect(() => {
    listAuditEvents({ limit: 200 })
      .then(setEvents)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el historial"))
      .finally(() => setLoading(false));
  }, []);

  const totalPages = Math.ceil(events.length / PAGE_SIZE);
  const pageEvents = events.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  return (
    <div className="p-4">
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <Activity className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Actividad del sistema
      </h1>

      {error && <Banner tone="danger">{error}</Banner>}

      {loading ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : events.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">No hay eventos registrados todavía.</p>
      ) : (
        <div className="flex flex-col gap-1">
          {/* Cabecera de columnas */}
          <div className="grid grid-cols-[140px_90px_110px_1fr_130px_160px] gap-3 px-3 py-1 text-xs font-medium text-(--text-muted)">
            <span>Acción</span>
            <span>Tipo</span>
            <span>Referencia</span>
            <span>Tercero</span>
            <span>Usuario</span>
            <span className="text-right">Fecha y hora</span>
          </div>

          {pageEvents.map((e) => {
            const label     = ACTION_LABELS[e.action] ?? e.action;
            const colorClass = ACTION_COLORS[e.action] ?? "text-(--text-secondary) bg-(--bg-secondary)";
            const docType   = eventDocType(e.metadata);
            const docRef    = eventDocRef(e.metadata);
            const party     = eventParty(e.metadata);
            const env       = eventEnv(e.metadata);
            const date      = new Date(e.created_at);
            const dateStr   = date.toLocaleDateString("es-CO", { day: "2-digit", month: "short", year: "numeric" });
            const timeStr   = date.toLocaleTimeString("es-CO", { hour: "2-digit", minute: "2-digit", hour12: false });
            const actor     = e.user_name || e.user_email || "Sistema";

            return (
              <div
                key={e.id}
                className="grid grid-cols-[140px_90px_110px_1fr_130px_160px] items-center gap-3 rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 text-xs"
              >
                {/* Acción */}
                <span className={`inline-flex w-fit items-center rounded px-1.5 py-0.5 font-medium ${colorClass}`}>
                  {label}
                </span>

                {/* Tipo de documento */}
                <span className="truncate text-(--text-secondary)">
                  {docType || <span className="text-(--text-muted)">—</span>}
                  {env && <span className="ml-1 text-(--text-muted)">· {env === "Habilitación" ? "HAB" : "PRD"}</span>}
                </span>

                {/* Referencia (prefix-número, clickeable) */}
                {e.resource_id ? (
                  <button
                    type="button"
                    onClick={() => navigate(`/documents/invoices/${e.resource_id}`)}
                    className="truncate text-left font-mono font-semibold text-(--accent-primary) hover:underline"
                  >
                    {docRef}
                  </button>
                ) : (
                  <span className="truncate font-mono text-(--text-muted)">{docRef}</span>
                )}

                {/* Tercero */}
                <span className="min-w-0 truncate text-(--text-secondary)">{party}</span>

                {/* Actor */}
                <span className="truncate text-(--text-muted)">{actor}</span>

                {/* Fecha · hora */}
                <span className="whitespace-nowrap text-right text-(--text-muted)">{dateStr} · {timeStr}</span>
              </div>
            );
          })}

          {/* Paginador */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-2 text-xs text-(--text-muted)">
              <span>
                {page * PAGE_SIZE + 1}–{Math.min((page + 1) * PAGE_SIZE, events.length)} de {events.length}
              </span>
              <div className="flex gap-1">
                <button
                  type="button"
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                  className="rounded border border-(--border-color) px-2 py-0.5 disabled:opacity-40 hover:not-disabled:bg-(--bg-secondary)"
                >
                  ← Anterior
                </button>
                <button
                  type="button"
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage((p) => p + 1)}
                  className="rounded border border-(--border-color) px-2 py-0.5 disabled:opacity-40 hover:not-disabled:bg-(--bg-secondary)"
                >
                  Siguiente →
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
