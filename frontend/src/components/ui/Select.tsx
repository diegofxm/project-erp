import type { SelectHTMLAttributes } from "react";

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  error?: string;
}

// Mismo patrón visual que Input.tsx (sección 11 del design system) — fondo bg-primary, borde
// 1px, rounded, sin estado de foco propio (lo cubre :focus-visible global en index.css).
export function Select({ label, error, className = "", id, required, children, ...rest }: SelectProps) {
  return (
    <label className="flex flex-col gap-1" htmlFor={id}>
      {label && (
        <span className="text-xs font-medium text-(--text-secondary)">
          {label}{required && <span className="ml-0.5 text-(--color-danger-text)">*</span>}
        </span>
      )}
      <select
        id={id}
        required={required}
        className={`w-full rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 text-xs text-(--text-primary) transition-colors sm:py-1.5 ${className}`}
        {...rest}
      >
        {children}
      </select>
      {error && <span className="text-xs text-(--color-danger-text)">{error}</span>}
    </label>
  );
}
