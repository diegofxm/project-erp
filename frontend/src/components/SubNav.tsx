import { NavLink, useLocation } from "react-router";
import { DOC_COLORS } from "../lib/docColors";

interface SubNavItem {
  to: string;
  label: string;
  color?: string;
}

interface SubNavConfig {
  prefix: string;
  items: SubNavItem[];
}

const SUB_NAVS: SubNavConfig[] = [
  {
    prefix: "/admin",
    items: [
      { to: "/admin/billing",   label: "Facturación" },
      { to: "/admin/renewals",  label: "Renovaciones" },
      { to: "/admin/issuer",    label: "Por emisor" },
      { to: "/admin/plans",     label: "Planes" },
      { to: "/admin/users",     label: "Usuarios" },
      { to: "/admin/prospects", label: "Solicitudes" },
    ],
  },
  {
    prefix: "/documents",
    items: [
      { to: "/documents/invoices",          label: "Factura Electrónica", color: DOC_COLORS["/documents/invoices"] },
      { to: "/documents/credit-notes",      label: "Nota Crédito",        color: DOC_COLORS["/documents/credit-notes"] },
      { to: "/documents/debit-notes",       label: "Nota Débito",         color: DOC_COLORS["/documents/debit-notes"] },
      { to: "/documents/support-documents", label: "Documento Soporte",   color: DOC_COLORS["/documents/support-documents"] },
      { to: "/documents/adjustment-notes",  label: "Nota de Ajuste",      color: DOC_COLORS["/documents/adjustment-notes"] },
    ],
  },
  {
    prefix: "/settings",
    items: [
      { to: "/settings/general",  label: "General" },
      { to: "/settings/account",  label: "Mi cuenta" },
      { to: "/settings/company",  label: "Empresa" },
      { to: "/settings/activity", label: "Actividad" },
    ],
  },
];

export function SubNav() {
  const { pathname } = useLocation();
  const config = SUB_NAVS.find((s) => pathname.startsWith(s.prefix));
  if (!config) return null;

  return (
    <nav className="flex h-10 shrink-0 overflow-x-auto border-b border-(--border-color) bg-(--bg-secondary) px-1">
      {config.items.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          className={({ isActive }) =>
            `inline-flex h-full shrink-0 items-center gap-1.5 border-b-2 px-3 text-xs font-medium transition-colors ${
              isActive
                ? `-mb-px ${item.color ? "border-current" : "border-(--accent-primary) text-(--accent-primary)"}`
                : "border-transparent text-(--text-secondary) hover:text-(--text-primary)"
            }`
          }
          style={({ isActive }) =>
            item.color && isActive ? { color: item.color } : undefined
          }
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  );
}
