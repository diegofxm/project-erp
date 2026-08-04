import { NavLink, useLocation } from "react-router";
import { DOC_COLORS } from "../lib/docColors";

interface SubNavItem {
  to: string;
  label: string;
  color?: string;
  // Solo hace falta cuando `to` es un prefijo de otro ítem de la misma sección (ej. "/sales"
  // es prefijo de "/sales/quotes") — sin esto, NavLink los marcaría activos a los dos a la vez.
  end?: boolean;
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
      { to: "/admin/company",   label: "Por empresa" },
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
    prefix: "/sales",
    items: [
      { to: "/sales", label: "Panel", end: true },
      { to: "/sales/quotes", label: "Cotizaciones" },
      { to: "/sales/orders", label: "Ventas" },
      { to: "/sales/receivables", label: "Cartera" },
    ],
  },
  {
    prefix: "/purchases",
    items: [
      { to: "/purchases", label: "Panel", end: true },
      { to: "/purchases/orders", label: "Compras" },
      { to: "/purchases/payables", label: "Cuentas por pagar" },
    ],
  },
  {
    prefix: "/inventory",
    items: [
      { to: "/inventory", label: "Panel", end: true },
      { to: "/inventory/stock", label: "Existencias" },
      { to: "/inventory/movements", label: "Movimientos" },
      { to: "/inventory/warehouses", label: "Bodegas" },
    ],
  },
  {
    prefix: "/accounting",
    items: [
      { to: "/accounting", label: "Panel", end: true },
      { to: "/accounting/journals", label: "Asientos" },
      { to: "/accounting/accounts", label: "Cuentas" },
      { to: "/accounting/periods", label: "Periodos" },
      { to: "/accounting/reports", label: "Reportes" },
      { to: "/accounting/bank", label: "Bancos" },
      { to: "/accounting/fixed-assets", label: "Activos fijos" },
      { to: "/accounting/budgets", label: "Presupuestos" },
      { to: "/accounting/declarations", label: "Declaraciones" },
      { to: "/accounting/certificates", label: "Certificados" },
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
          end={item.end}
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
