import type { DocumentStatus } from "../../lib/types";

const LABELS: Record<DocumentStatus, string> = {
  draft: "Borrador",
  built: "Construido",
  sent: "Enviado",
  accepted: "Aceptado",
  rejected: "Rechazado",
  send_error: "Error de envío",
};

// Mismos tokens pastel que Banner.tsx (sección 2.3 del design system) — draft/built son
// neutros (todavía no se decide nada ante la DIAN), sent es informativo, accepted es éxito,
// rejected/send_error son error.
const TONE_CLASSES: Record<DocumentStatus, string> = {
  draft: "bg-(--bg-tertiary) text-(--text-secondary)",
  built: "bg-(--bg-tertiary) text-(--text-secondary)",
  sent: "bg-(--color-info-bg) text-(--color-info-text)",
  accepted: "bg-(--color-success-bg) text-(--color-success-text)",
  rejected: "bg-(--color-danger-bg) text-(--color-danger-text)",
  send_error: "bg-(--color-danger-bg) text-(--color-danger-text)",
};

export function StatusBadge({ status }: { status: DocumentStatus }) {
  return <span className={`rounded px-2 py-0.5 text-xs font-medium ${TONE_CLASSES[status]}`}>{LABELS[status]}</span>;
}
