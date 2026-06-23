import type { HTMLAttributes } from "react";

// Contenedor genérico tipo tarjeta (login, register, onboarding, paneles del dashboard).
// rounded-lg reservado para contenedores grandes (recomendación #6 del design system) — los
// elementos chicos (botones, inputs, badges) usan "rounded" (4px), nunca este componente.
export function Card({ className = "", children, ...rest }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={`rounded-lg border border-(--border-light) bg-(--bg-secondary) shadow-xl ${className}`}
      {...rest}
    >
      {children}
    </div>
  );
}
