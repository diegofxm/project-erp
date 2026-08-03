import { useEffect, useMemo, useState } from "react";
import { Landmark, Search } from "lucide-react";
import { listAccounts } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import type { Account } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { InfoTip } from "../components/ui/InfoTip";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

const CATEGORIES = ["Activo", "Pasivo", "Patrimonio", "Ingreso", "Gasto", "Costo", "Costo de Producción", "Cuenta de Orden Deudora", "Cuenta de Orden Acreedora"];

const CATEGORY_TONE: Record<string, StatusTone> = {
  Activo: "info", Pasivo: "warning", Patrimonio: "neutral",
  Ingreso: "success", Gasto: "danger", Costo: "danger", "Costo de Producción": "danger",
};

const MAX_RESULTS = 300;

export function AccountingAccountsPage() {
  const [accounts, setAccounts] = useState<Account[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");

  useEffect(() => {
    listAccounts()
      .then(setAccounts)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el catálogo de cuentas"));
  }, []);

  const filtered = useMemo(() => {
    if (!accounts) return [];
    const q = search.trim().toLowerCase();
    return accounts
      .filter((a) => {
        if (category && a.category !== category) return false;
        if (q && !a.code.startsWith(q) && !a.name.toLowerCase().includes(q)) return false;
        return true;
      })
      .sort((a, b) => a.code.localeCompare(b.code));
  }, [accounts, search, category]);

  const active = search.trim() !== "" || category !== "";
  const shown = filtered.slice(0, MAX_RESULTS);

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Cuentas" }]} />
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Landmark className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Catálogo de cuentas (PUC)
          <InfoTip>
            Plan Único de Cuentas (Decreto 2650) — catálogo estándar colombiano, compartido por todas las
            empresas de la plataforma. Solo las cuentas marcadas <strong>de movimiento</strong> aceptan
            asientos directamente; las demás son cuentas de agrupación (mayor, subcuenta).
          </InfoTip>
        </h1>
        <div className="flex flex-wrap items-center gap-1.5">
          <div className="relative">
            <Search className="pointer-events-none absolute left-2 top-1/2 h-3 w-3 -translate-y-1/2 text-(--text-muted)" />
            <input
              type="search"
              placeholder="Código o nombre..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="rounded border border-(--border-color) bg-(--bg-primary) py-1 pl-6 pr-2 text-xs text-(--text-primary) transition-colors w-56"
            />
          </div>
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1 text-xs text-(--text-primary) transition-colors"
          >
            <option value="">Todas las categorías</option>
            {CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {accounts === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : !active ? (
        <p className="text-xs text-(--text-secondary)">
          El catálogo tiene {accounts.length.toLocaleString("es-CO")} cuentas — escribe un código/nombre o elige una categoría para buscar.
        </p>
      ) : shown.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">No hay cuentas que coincidan con la búsqueda.</p>
      ) : (
        <>
          <div className="overflow-x-auto rounded border border-(--border-color)">
            <table className="w-full text-left text-xs">
              <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                <tr>
                  <th className="px-3 py-2 font-medium">Código</th>
                  <th className="px-3 py-2 font-medium">Nombre</th>
                  <th className="px-3 py-2 font-medium">Categoría</th>
                  <th className="px-3 py-2 font-medium">Nivel</th>
                  <th className="px-3 py-2 font-medium">Tipo</th>
                </tr>
              </thead>
              <tbody>
                {shown.map((a, i) => (
                  <tr key={a.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                    <td className="px-3 py-2 font-mono text-(--text-primary)" style={{ paddingLeft: `${(a.level - 1) * 12 + 12}px` }}>{a.code}</td>
                    <td className="px-3 py-2 text-(--text-primary)">{a.name}</td>
                    <td className="px-3 py-2"><StatusPill tone={CATEGORY_TONE[a.category] ?? "neutral"} label={a.category} /></td>
                    <td className="px-3 py-2 text-(--text-secondary)">{a.level}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{a.is_posting ? "De movimiento" : "Agrupación"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {filtered.length > MAX_RESULTS && (
            <p className="mt-2 text-xs text-(--text-secondary)">
              Mostrando los primeros {MAX_RESULTS} de {filtered.length} resultados — afina la búsqueda para ver menos.
            </p>
          )}
        </>
      )}
    </div>
  );
}
