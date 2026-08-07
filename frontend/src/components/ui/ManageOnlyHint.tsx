import type { ReactNode } from "react";

// Nota que reemplaza (no acompaña) un control de gestión oculto por useCanManage — el patrón es
// ocultar el control por completo para quien no puede usarlo, nunca dejarlo visible-pero-atenuado
// (eso es un patrón de upsell de SaaS, no el que usan ERPs como Odoo o SIESA: ahí quien no tiene
// permiso simplemente no ve el botón de edición). Mensaje centralizado acá para que no se
// desincronice la redacción entre las secciones que lo usan — children permite un mensaje
// contextual puntual (ej. "sin logo — solo un administrador puede subir uno").
export function ManageOnlyHint({ children }: { children?: ReactNode }) {
  return (
    <p className="text-xs text-(--text-muted)">
      {children ?? "Solo un administrador o dueño de la empresa puede editar esto."}
    </p>
  );
}
