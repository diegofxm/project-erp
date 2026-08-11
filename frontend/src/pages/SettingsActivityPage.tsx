import { useEffect, useState } from "react";
import { Activity } from "lucide-react";
import { useNavigate } from "react-router";
import { listAuditEvents } from "../lib/audit";
import type { AuditEvent } from "../lib/types";
import { ApiError } from "../lib/apiClient";
import { Banner } from "../components/ui/Banner";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";

// Espejo legible de cada acción de auditoría que el backend realmente registra (ver
// h.logAudit/h.audit.Log en cada módulo). No queda ninguna acción sin traducir: las que no están
// aquí caen en humanizeAction(), que nunca devuelve el código crudo (ej. "sale.confirmed").
const ACTION_LABELS: Record<string, string> = {
  // Autenticación
  "auth.login":            "Inicio de sesión",
  "auth.logout":           "Cierre de sesión",
  "auth.invite_accepted":  "Invitación aceptada",
  "auth.company_selected": "Empresa seleccionada",
  "auth.profile_updated":  "Perfil actualizado",
  "auth.password_changed": "Contraseña cambiada",
  "auth.user_invited":     "Usuario invitado",
  // Empresa
  "company.profile_updated":     "Datos de la empresa actualizados",
  "company.credentials_updated": "Credenciales DIAN actualizadas",
  "company.credentials_cleared": "Credenciales DIAN eliminadas",
  "company.logo_updated":        "Logo actualizado",
  "company.logo_deleted":        "Logo eliminado",
  // Terceros (clientes/proveedores)
  "customer.created":       "Cliente creado",
  "customer.updated":       "Cliente actualizado",
  "customer.role_removed":  "Rol de cliente retirado",
  "customer.data_exported": "Datos de cliente exportados",
  "supplier.created":       "Proveedor creado",
  "supplier.updated":       "Proveedor actualizado",
  "supplier.role_removed":  "Rol de proveedor retirado",
  "supplier.data_exported": "Datos de proveedor exportados",
  // Productos e inventario
  "product.created":              "Producto creado",
  "product.updated":              "Producto actualizado",
  "product.deleted":              "Producto eliminado",
  "inventory.movement_created":   "Movimiento de inventario",
  "inventory.movement_deleted":   "Movimiento eliminado",
  // Ventas
  "sale.updated":          "Venta actualizada",
  "sale.confirmed":        "Venta confirmada",
  "sale.cancelled":        "Venta cancelada",
  "sale.deleted":          "Venta eliminada",
  "sale.payment_recorded": "Pago de venta registrado",
  // Compras
  "purchase.updated":   "Compra actualizada",
  "purchase.confirmed": "Compra confirmada",
  "purchase.received":  "Compra recibida",
  // Contabilidad
  "declaration.iva_filed":         "Declaración de IVA presentada",
  "declaration.income_tax_filed":  "Declaración de renta presentada",
  "declaration.ica_filed":         "Declaración de ICA presentada",
  "fixed_asset.created":           "Activo fijo creado",
  "fixed_asset.disposed":          "Activo fijo dado de baja",
  "depreciation.run":              "Depreciación ejecutada",
  "journal.posted":                "Comprobante contabilizado",
  "journal.voided":                "Comprobante anulado",
  "period.closed":                 "Periodo cerrado",
  "period.reopened":               "Periodo reabierto",
  "reconciliation.marked":         "Conciliación marcada",
  "reconciliation.unmarked":       "Conciliación desmarcada",
  // Nómina y RRHH
  "employee.created":     "Empleado creado",
  "employee.updated":     "Empleado actualizado",
  "employee.deactivated": "Empleado desactivado",
  "payroll.generated":    "Nómina generada",
  "absence.requested":    "Ausencia solicitada",
  "absence.updated":      "Ausencia actualizada",
  "absence.withdrawn":    "Ausencia retirada",
  "absence.approved":     "Ausencia aprobada",
  "absence.rejected":     "Ausencia rechazada",
  // Documentos electrónicos
  "document.created":            "Borrador creado",
  "document.updated":            "Borrador actualizado",
  "document.confirmed":          "Documento confirmado",
  "document.status_checked":     "Estado verificado ante la DIAN",
  "document.deleted":            "Eliminado",
  "document.email_sent":         "Correo enviado",
  "document.cloned":             "Clonado",
  "numbering_range.created":     "Rango de numeración creado",
  "numbering_range.deactivated": "Rango de numeración desactivado",
  "numbering_range.activated":   "Rango de numeración reactivado",
};

// Verbos que indican intención (última palabra de la acción, sin importar el prefijo) -- evita
// mantener un mapa de color por cada una de las ~50 acciones de arriba. Cualquier acción nueva
// que el backend agregue ya cae en el color correcto solo por cómo termina su nombre.
const VERB_TONE: Record<string, string> = {
  created: "success", requested: "success", invited: "success", recorded: "success",
  generated: "success", posted: "success", filed: "success", marked: "success",
  approved: "success", confirmed: "success", received: "success", activated: "success",
  accepted: "success", selected: "success", run: "success",
  deleted: "danger", cancelled: "danger", rejected: "danger", voided: "danger",
  deactivated: "danger", removed: "danger", withdrawn: "danger", cleared: "danger",
  unmarked: "danger", logout: "danger",
  updated: "neutral", checked: "neutral", changed: "neutral",
  sent: "accent", cloned: "accent", exported: "accent",
};

const TONE_CLASSES: Record<string, string> = {
  success: "text-(--color-success-text) bg-(--color-success-bg)",
  danger:  "text-(--color-danger-text) bg-(--color-danger-bg)",
  neutral: "text-(--text-secondary) bg-(--bg-secondary)",
  accent:  "text-(--accent-primary) bg-(--bg-secondary)",
  info:    "text-(--color-info-text) bg-(--color-info-bg)",
};

// humanizeAction traduce cualquier código de acción a texto legible -- primero busca la
// traducción curada; si no existe (acción nueva que el backend agregó y este archivo aún no
// conoce), arma un texto a partir del código en vez de mostrarlo crudo: "fixed_asset.created" ->
// "Fixed asset created" en el peor de los casos, nunca un identificador con puntos y guiones bajos.
function humanizeAction(action: string): string {
  const curated = ACTION_LABELS[action];
  if (curated) return curated;
  const readable = action.replace(/[._]/g, " ").trim();
  return readable.charAt(0).toUpperCase() + readable.slice(1);
}

function actionToneClass(action: string): string {
  const verb = action.split(/[._]/).pop() ?? "";
  const tone = VERB_TONE[verb] ?? "neutral";
  return TONE_CLASSES[tone];
}

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
  return ((meta.customer_name ?? meta.supplier_name ?? "") as string) || "—";
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
      <Breadcrumbs items={[{ label: "Configuración", to: "/settings/general" }, { label: "Actividad" }]} />
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
        <div className="flex flex-col gap-2">
        <div className="flex flex-col overflow-hidden rounded border border-(--border-color)">
          {/* Cabecera de columnas -- formato log: lista continua, no una tarjeta por evento */}
          <div className="grid grid-cols-[150px_90px_110px_1fr_130px_150px] gap-3 border-b border-(--border-color) bg-(--bg-tertiary) px-3 py-1.5 font-mono text-[11px] font-medium uppercase tracking-wide text-(--text-muted)">
            <span>Acción</span>
            <span>Tipo</span>
            <span>Referencia</span>
            <span>Tercero</span>
            <span>Usuario</span>
            <span className="text-right">Fecha · hora</span>
          </div>

          {pageEvents.map((e) => {
            const label      = humanizeAction(e.action);
            const toneClass  = actionToneClass(e.action);
            const docType    = eventDocType(e.metadata);
            const docRef     = eventDocRef(e.metadata);
            const party      = eventParty(e.metadata);
            const env        = eventEnv(e.metadata);
            const date       = new Date(e.created_at);
            const dateStr    = date.toLocaleDateString("es-CO", { day: "2-digit", month: "short", year: "numeric" });
            const timeStr    = date.toLocaleTimeString("es-CO", { hour: "2-digit", minute: "2-digit", hour12: false });
            const actor      = e.user_name || e.user_email || "Sistema";

            return (
              <div
                key={e.id}
                className="grid grid-cols-[150px_90px_110px_1fr_130px_150px] items-center gap-3 border-b border-(--border-light) px-3 py-1.5 font-mono text-[11px] last:border-b-0 hover:bg-(--bg-secondary)"
              >
                {/* Acción */}
                <span className={`inline-flex w-fit items-center rounded px-1.5 py-0.5 font-medium ${toneClass}`} title={e.action}>
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
                    className="truncate text-left font-semibold text-(--accent-primary) hover:underline"
                  >
                    {docRef}
                  </button>
                ) : (
                  <span className="truncate text-(--text-muted)">{docRef}</span>
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
        </div>

          {/* Paginador */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between px-1 text-xs text-(--text-muted)">
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
