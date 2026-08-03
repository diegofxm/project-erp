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

const LINK_CLASSES = "text-(--accent-primary) hover:underline";

// Miga de pan simple, estilo Odoo: trail muted con el ítem actual resaltado.
// El último ítem nunca es interactivo (es la página en la que ya estás).
// Si solo hay un ítem (nada se desprende de una sección principal — ej. "Inicio",
// "Clientes" en su vista normal) no se muestra nada; el sidebar ya marca la ubicación.
export function Breadcrumbs({ items }: { items: BreadcrumbItem[] }) {
  if (items.length < 2) return null;

  return (
    <nav className="mb-1 flex items-center gap-1 text-xs text-(--text-muted)">
      {items.map((item, i) => {
        const isLast = i === items.length - 1;
        return (
          <span key={i} className="flex items-center gap-1">
            {i > 0 && <ChevronRight className="h-3 w-3 shrink-0 opacity-60" />}
            {!isLast && item.to ? (
              <Link to={item.to} className={LINK_CLASSES}>
                {item.label}
              </Link>
            ) : !isLast && item.onClick ? (
              <button type="button" onClick={item.onClick} className={LINK_CLASSES}>
                {item.label}
              </button>
            ) : (
              <span className={isLast ? "font-medium text-(--accent-primary)" : undefined}>{item.label}</span>
            )}
          </span>
        );
      })}
    </nav>
  );
}
