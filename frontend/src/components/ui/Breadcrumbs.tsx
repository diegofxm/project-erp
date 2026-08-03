import { Link } from "react-router";
import { ChevronRight } from "lucide-react";

export interface BreadcrumbItem {
  label: string;
  to?: string;
}

// Miga de pan simple, estilo Odoo: trail muted con el ítem actual resaltado.
// El último ítem nunca es link (es la página en la que ya estás), los anteriores sí.
export function Breadcrumbs({ items }: { items: BreadcrumbItem[] }) {
  return (
    <nav className="mb-1 flex items-center gap-1 text-xs text-(--text-muted)">
      {items.map((item, i) => {
        const isLast = i === items.length - 1;
        return (
          <span key={i} className="flex items-center gap-1">
            {i > 0 && <ChevronRight className="h-3 w-3 shrink-0 opacity-60" />}
            {item.to && !isLast ? (
              <Link to={item.to} className="hover:text-(--text-primary)">
                {item.label}
              </Link>
            ) : (
              <span className={isLast ? "font-medium text-(--accent-primary)" : undefined}>{item.label}</span>
            )}
          </span>
        );
      })}
    </nav>
  );
}
