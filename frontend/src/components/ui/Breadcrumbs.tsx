import { Link } from "react-router";
import { ChevronRight } from "lucide-react";

export interface BreadcrumbItem {
  label: string;
  /** Navega a otra ruta real (react-router). No sirve si ya estás en esa misma URL. */
  to?: string;
  /** Para volver dentro de la MISMA ruta (ej. cerrar un formulario inline como "Nuevo cliente",
   * que no tiene URL propia) — usar esto en vez de `to` cuando no hay cambio de ruta real. */
  onClick?: () => void;
}

// Subrayado siempre visible (no solo al hover) para que un ítem clicable se note
// como link a simple vista, sin depender del color para distinguirlo del actual.
const LINK_CLASSES = "text-(--accent-primary) underline decoration-(--accent-primary)/40 underline-offset-2 hover:text-(--accent-hover) hover:decoration-(--accent-hover)";

// Único componente de migas de pan de la app — no duplicar este patrón a mano en páginas.
// Estilo Odoo: los pasos anteriores son links subrayados, el ítem actual es texto plano
// sin subrayar (es donde ya estás, no es clicable). Si un paso intermedio no tiene `to` ni
// `onClick` (ej. "Documentos", que no tiene una sub-página por defecto), se muestra como
// texto plano sin color de link — sigue dando contexto sin prometer un click que no hace nada.
// Si solo hay un ítem (nada se desprende de una sección principal — ej. "Inicio",
// "Clientes" en su vista normal) no se muestra nada; el sidebar ya marca la ubicación.
export function Breadcrumbs({ items }: { items: BreadcrumbItem[] }) {
  if (items.length < 2) return null;

  return (
    <nav className="mb-1 flex items-center gap-1 text-xs">
      {items.map((item, i) => {
        const isLast = i === items.length - 1;
        return (
          <span key={i} className="flex items-center gap-1">
            {i > 0 && <ChevronRight className="h-3 w-3 shrink-0 text-(--text-muted) opacity-60" />}
            {!isLast && item.to ? (
              <Link to={item.to} className={LINK_CLASSES}>
                {item.label}
              </Link>
            ) : !isLast && item.onClick ? (
              <button type="button" onClick={item.onClick} className={LINK_CLASSES}>
                {item.label}
              </button>
            ) : (
              <span className={isLast ? "font-medium text-(--text-primary)" : "text-(--text-muted)"}>
                {item.label}
              </span>
            )}
          </span>
        );
      })}
    </nav>
  );
}
