import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Spinner } from "./Spinner";

// Taxonomía de botones del design system (sección 10): 5 variantes, sin componente
// reutilizable en el proyecto original — esto corrige justamente esa falta de aquí en
// adelante. Hover vía clases Tailwind (hover:), nunca con onMouseEnter/onMouseLeave.
type Variant = "primary" | "success" | "danger" | "secondary" | "ghost";

const VARIANT_CLASSES: Record<Variant, string> = {
  primary: "bg-(--accent-primary) text-white hover:bg-(--accent-hover)",
  success: "bg-(--color-success) text-white hover:bg-(--color-success-hover)",
  danger: "bg-(--color-danger) text-white hover:bg-(--color-danger-hover)",
  secondary: "bg-(--bg-tertiary) text-(--text-primary) hover:bg-(--bg-hover)",
  ghost: "bg-transparent text-(--text-secondary) hover:bg-(--bg-hover)",
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  icon?: ReactNode;
  loading?: boolean;
}

export function Button({ variant = "primary", icon, loading, children, className = "", disabled, ...rest }: ButtonProps) {
  return (
    <button
      className={`inline-flex items-center justify-center gap-1.5 rounded px-3 py-2 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 sm:py-1.5 ${VARIANT_CLASSES[variant]} ${className}`}
      disabled={disabled || loading}
      {...rest}
    >
      {loading ? <Spinner className="h-3.5 w-3.5" /> : icon}
      {children}
    </button>
  );
}
