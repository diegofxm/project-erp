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
  "document.confirmed":  "Documento confirmado",
  "document.deleted":    "Borrador eliminado",
  "document.email_sent": "Correo enviado",
  "document.cloned":     "Documento clonado",
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
  "01": "FE",
  "91": "NC",
  "92": "ND",
  "05": "DS",
  "95": "NA",
};

function eventParty(meta?: Record<string, unknown>): string {
  if (!meta) return "";
  const name = (meta.customer_name ?? meta.vendor_name ?? "") as string;
  return name;
}

function eventDocLabel(meta?: Record<string, unknown>): string {
  if (!meta) return "";
  const typeCode = meta.dian_document_type_code as string | undefined;
  const prefix = meta.prefix as string | undefined;
  const number = meta.number as number | undefined;
  if (!typeCode) return "";
  const type = DOC_TYPE_LABELS[typeCode] ?? typeCode;
  if (prefix || number) return `${type} ${prefix ?? ""}${number ?? ""}`;
  return type;
}

export function SettingsActivityPage() {
  const navigate = useNavigate();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listAuditEvents({ limit: 100 })
      .then(setEvents)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el historial"))
      .finally(() => setLoading(false));
  }, []);

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
          {events.map((e) => {
            const label = ACTION_LABELS[e.action] ?? e.action;
            const colorClass = ACTION_COLORS[e.action] ?? "text-(--text-secondary) bg-(--bg-secondary)";
            const party = eventParty(e.metadata);
            const docLabel = eventDocLabel(e.metadata);
            const date = new Date(e.created_at);
            const dateStr = date.toLocaleDateString("es-CO", { day: "2-digit", month: "short", year: "numeric" });
            const timeStr = date.toLocaleTimeString("es-CO", { hour: "2-digit", minute: "2-digit" });

            return (
              <div
                key={e.id}
                className="flex items-start gap-3 rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 text-xs"
              >
                <span className={`mt-0.5 shrink-0 rounded px-1.5 py-0.5 font-medium ${colorClass}`}>
                  {label}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-baseline gap-x-2">
                    {docLabel && (
                      <button
                        type="button"
                        disabled={!e.resource_id}
                        onClick={() => e.resource_id && navigate(`/documents/invoices/${e.resource_id}`)}
                        className="font-medium text-(--text-primary) hover:underline disabled:cursor-default disabled:no-underline"
                      >
                        {docLabel}
                      </button>
                    )}
                    {party && <span className="text-(--text-secondary)">{party}</span>}
                  </div>
                  <div className="mt-0.5 flex flex-wrap gap-x-3 text-(--text-muted)">
                    <span>{e.user_name || e.user_email || "Sistema"}</span>
                    <span>{dateStr} · {timeStr}</span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
