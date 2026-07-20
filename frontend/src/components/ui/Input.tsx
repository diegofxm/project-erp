import type { InputHTMLAttributes } from "react";

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

// Sección 11 del design system: fondo bg-primary (más "hundido" que la tarjeta que lo
// contiene), borde 1px, rounded, sin estado de foco propio — el foco global ya lo cubre
// index.css (:focus-visible), no hace falta repetirlo aquí (recomendación #4).
export function Input({ label, error, className = "", id, required, ...rest }: InputProps) {
  return (
    <label className="flex flex-col gap-1" htmlFor={id}>
      {label && (
        <span className="text-xs font-medium text-(--text-secondary)">
          {label}{required && <span className="ml-0.5 text-(--color-danger-text)">*</span>}
        </span>
      )}
      <input
        id={id}
        required={required}
        className={`w-full rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 text-xs text-(--text-primary) placeholder:text-(--text-muted) transition-colors sm:py-1.5 ${className}`}
        {...rest}
      />
      {error && <span className="text-xs text-(--color-danger-text)">{error}</span>}
    </label>
  );
}
