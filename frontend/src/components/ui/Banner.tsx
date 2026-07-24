import { AlertCircle, CheckCircle2, Info } from "lucide-react";
import type { ReactNode } from "react";

type Tone = "success" | "danger" | "info";

const TONE_CLASSES: Record<Tone, string> = {
  success: "bg-(--color-success-bg) border-(--color-success-border) text-(--color-success-text)",
  danger: "bg-(--color-danger-bg) border-(--color-danger-border) text-(--color-danger-text)",
  info: "bg-(--color-info-bg) border-(--color-info-border) text-(--color-info-text)",
};

const TONE_ICONS: Record<Tone, ReactNode> = {
  success: <CheckCircle2 className="h-4 w-4 shrink-0" />,
  danger: <AlertCircle className="h-4 w-4 shrink-0" />,
  info: <Info className="h-4 w-4 shrink-0" />,
};

// Banner pastel fijo por tipo (sección 2.3 del design system) — para errores de login/register
// y mensajes de estado del onboarding.
export function Banner({ tone, children }: { tone: Tone; children: ReactNode }) {
  return (
    <div className={`flex items-start gap-2 rounded border px-3 py-2.5 text-xs ${TONE_CLASSES[tone]}`}>
      {TONE_ICONS[tone]}
      <span>{children}</span>
    </div>
  );
}
