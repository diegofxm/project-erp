import { NavLink, useLocation } from "react-router";

interface SubNavItem {
  to: string;
  label: string;
}

interface SubNavConfig {
  prefix: string;
  items: SubNavItem[];
}

// Mapa de prefijos → pestañas. Agregar aquí cuando una sección necesite sub-navegación
// (ej. /settings con sus propias sub-páginas a futuro) sin tocar el Sidebar.
const SUB_NAVS: SubNavConfig[] = [
  {
    prefix: "/documents",
    items: [
      { to: "/documents/invoices", label: "Factura Electrónica" },
      { to: "/documents/credit-notes", label: "Nota Crédito" },
      { to: "/documents/debit-notes", label: "Nota Débito" },
    ],
  },
  {
    prefix: "/settings",
    items: [
      { to: "/settings/general", label: "General" },
      { to: "/settings/company", label: "Empresa" },
    ],
  },
];

export function SubNav() {
  const { pathname } = useLocation();
  const config = SUB_NAVS.find((s) => pathname.startsWith(s.prefix));
  if (!config) return null;

  return (
    <nav className="flex h-10 shrink-0 border-b border-(--border-color) bg-(--bg-secondary) px-3">
      {config.items.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          className={({ isActive }) =>
            `inline-flex h-full items-center border-b-2 px-3 text-xs font-medium transition-colors ${
              isActive
                ? "-mb-px border-(--accent-primary) text-(--accent-primary)"
                : "border-transparent text-(--text-secondary) hover:text-(--text-primary)"
            }`
          }
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  );
}
