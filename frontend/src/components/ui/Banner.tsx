import type { ReactNode } from "react";

type Tone = "success" | "danger" | "info";

const TONE_CLASSES: Record<Tone, string> = {
  success: "bg-(--color-success-bg) border-(--color-success-border) text-(--color-success-text)",
  danger: "bg-(--color-danger-bg) border-(--color-danger-border) text-(--color-danger-text)",
  info: "bg-(--color-info-bg) border-(--color-info-border) text-(--color-info-text)",
};

// Banner pastel fijo por tipo (sección 2.3 del design system) — para errores de login/register
// y mensajes de estado del onboarding.
export function Banner({ tone, children }: { tone: Tone; children: ReactNode }) {
  return <div className={`rounded border px-3 py-2 text-xs ${TONE_CLASSES[tone]}`}>{children}</div>;
}
