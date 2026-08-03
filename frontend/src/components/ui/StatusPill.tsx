// Misma píldora con punto de color que StatusBadge.tsx, pero genérica — StatusBadge queda
// específico de DocumentStatus (factura electrónica); acá cualquier módulo nuevo (ventas,
// cotizaciones, compras...) pasa su propio tono/etiqueta sin repetir las clases.
export type StatusTone = "neutral" | "info" | "success" | "warning" | "danger";

const TONE_CLASSES: Record<StatusTone, string> = {
  neutral: "bg-(--bg-tertiary) text-(--text-secondary)",
  info: "bg-(--color-info-bg) text-(--color-info-text)",
  success: "bg-(--color-success-bg) text-(--color-success-text)",
  warning: "bg-(--color-warning-bg) text-(--color-warning-text)",
  danger: "bg-(--color-danger-bg) text-(--color-danger-text)",
};

const DOT_CLASSES: Record<StatusTone, string> = {
  neutral: "bg-(--text-secondary)",
  info: "bg-(--color-info)",
  success: "bg-(--color-success)",
  warning: "bg-(--color-warning)",
  danger: "bg-(--color-danger)",
};

export function StatusPill({ tone, label }: { tone: StatusTone; label: string }) {
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${TONE_CLASSES[tone]}`}>
      <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${DOT_CLASSES[tone]}`} />
      {label}
    </span>
  );
}
