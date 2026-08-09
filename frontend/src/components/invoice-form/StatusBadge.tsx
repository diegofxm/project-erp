import type { DocumentStatus } from "../../lib/types";

const LABELS: Record<DocumentStatus, string> = {
  draft: "Borrador",
  built: "Construido",
  sent: "Enviado",
  accepted: "Aceptado",
  rejected: "Rechazado",
  send_error: "Error de envío",
  send_unknown: "Sin confirmar (revisar)",
};

// Mismos tokens pastel que Banner.tsx (sección 2.3 del design system) — draft/built son
// neutros (todavía no se decide nada ante la DIAN), sent es informativo, accepted es éxito,
// rejected/send_error son error. send_unknown usa el tono warning (no error): significa que
// no se pudo confirmar si la DIAN procesó el documento (timeout/conexión), no que
// definitivamente falló — por eso el consecutivo tampoco se libera automáticamente, requiere
// verificación manual antes de reintentar (ver domain.StatusSendUnknown en el backend).
const TONE_CLASSES: Record<DocumentStatus, string> = {
  draft: "bg-(--bg-tertiary) text-(--text-secondary)",
  built: "bg-(--bg-tertiary) text-(--text-secondary)",
  sent: "bg-(--color-info-bg) text-(--color-info-text)",
  accepted: "bg-(--color-success-bg) text-(--color-success-text)",
  rejected: "bg-(--color-danger-bg) text-(--color-danger-text)",
  send_error: "bg-(--color-danger-bg) text-(--color-danger-text)",
  send_unknown: "bg-(--color-warning-bg) text-(--color-warning-text)",
};

const DOT_CLASSES: Record<DocumentStatus, string> = {
  draft: "bg-(--text-secondary)",
  built: "bg-(--text-secondary)",
  sent: "bg-(--color-info)",
  accepted: "bg-(--color-success)",
  rejected: "bg-(--color-danger)",
  send_error: "bg-(--color-danger)",
  send_unknown: "bg-(--color-warning)",
};

// Píldora completa (rounded-full) con punto de color — patrón de estado tipo Odoo,
// en vez del rectángulo "rounded" (4px) plano que tenía antes.
export function StatusBadge({ status }: { status: DocumentStatus }) {
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${TONE_CLASSES[status]}`}>
      <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${DOT_CLASSES[status]}`} />
      {LABELS[status]}
    </span>
  );
}
