import type { SelectHTMLAttributes } from "react";

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  error?: string;
}

// Mismo patrón visual que Input.tsx (sección 11 del design system) — fondo bg-primary, borde
// 1px, rounded, sin estado de foco propio (lo cubre :focus-visible global en index.css).
export function Select({ label, error, className = "", id, children, ...rest }: SelectProps) {
  return (
    <label className="flex flex-col gap-1" htmlFor={id}>
      {label && <span className="text-xs font-medium text-(--text-secondary)">{label}</span>}
      <select
        id={id}
        className={`w-full rounded border border-(--border-color) bg-(--bg-primary) px-3 py-1.5 text-xs text-(--text-primary) transition-colors ${className}`}
        {...rest}
      >
        {children}
      </select>
      {error && <span className="text-xs text-(--color-danger-text)">{error}</span>}
    </label>
  );
}
