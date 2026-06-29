import { useEffect, useState } from "react";
import { CheckCircle2, X, XCircle } from "lucide-react";

export type ToastTone = "success" | "danger";

export interface ToastItem {
  id: number;
  tone: ToastTone;
  message: string;
}

// Mismos tokens de color que Banner.tsx (sección 2.3 del design system) — un toast es solo un
// Banner que aparece, se lee, y desaparece solo (ver ToastContext.tsx para el tiempo).
const TONE_CLASSES: Record<ToastTone, string> = {
  success: "bg-(--color-success-bg) border-(--color-success-border) text-(--color-success-text)",
  danger: "bg-(--color-danger-bg) border-(--color-danger-border) text-(--color-danger-text)",
};

const TONE_ICON: Record<ToastTone, typeof CheckCircle2> = {
  success: CheckCircle2,
  danger: XCircle,
};

export function Toast({ tone, message, onClose }: { tone: ToastTone; message: string; onClose: () => void }) {
  // mounted arranca en false para que la primera pintura sea con las clases "afuera" — al
  // siguiente tick pasa a true y la transición de Tailwind anima la entrada (translate-x +
  // opacity), sin librería de animación nueva.
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    const id = requestAnimationFrame(() => setMounted(true));
    return () => cancelAnimationFrame(id);
  }, []);

  const Icon = TONE_ICON[tone];

  return (
    <div
      className={`flex w-80 items-start gap-2 rounded border px-3 py-2 text-xs shadow-lg transition-all duration-300 ${TONE_CLASSES[tone]} ${
        mounted ? "translate-x-0 opacity-100" : "translate-x-4 opacity-0"
      }`}
    >
      <Icon className="mt-0.5 h-3.5 w-3.5 flex-shrink-0" />
      <p className="flex-1">{message}</p>
      <button type="button" onClick={onClose} className="flex-shrink-0 opacity-70 hover:opacity-100">
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
