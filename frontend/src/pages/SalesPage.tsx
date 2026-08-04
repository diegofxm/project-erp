import { useEffect, useMemo, useState } from "react";
import { ShoppingCart, Plus, Search } from "lucide-react";
import { useNavigate } from "react-router";
import { listSales } from "../lib/sales";
import { listCustomers } from "../lib/customers";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import type { Sale, SaleStatus, Customer } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Spinner } from "../components/ui/Spinner";
import { Pagination } from "../components/ui/Pagination";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

const PAGE_SIZE = 10;

const STATUS_LABEL: Record<SaleStatus, string> = { draft: "Borrador", confirmed: "Confirmada", cancelled: "Cancelada" };
const STATUS_TONE: Record<SaleStatus, StatusTone> = { draft: "neutral", confirmed: "success", cancelled: "danger" };

export function SalesPage() {
  const [sales, setSales] = useState<Sale[] | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<SaleStatus | "">("");
  const [search, setSearch] = useState("");
  const [offset, setOffset] = useState(0);
  const navigate = useNavigate();

  useEffect(() => {
    listSales()
      .then(setSales)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar las ventas"));
    listCustomers().then(setCustomers).catch(() => setCustomers([]));
  }, []);

  const customerName = useMemo(() => {
    const map = new Map(customers.map((c) => [c.id, c.name]));
    return (id: string) => map.get(id) ?? "—";
  }, [customers]);

  const filtered = useMemo(() => {
    if (!sales) return [];
    return sales.filter((s) => {
      if (status && s.status !== status) return false;
      if (search.trim()) {
        const q = search.toLowerCase();
        const matchesNumber = s.number?.toLowerCase().includes(q);
        const matchesCustomer = customerName(s.customer_id).toLowerCase().includes(q);
        if (!matchesNumber && !matchesCustomer) return false;
      }
      return true;
    });
  }, [sales, status, search, customerName]);

  const page = filtered.slice(offset, offset + PAGE_SIZE);
  const hasNext = offset + PAGE_SIZE < filtered.length;
  const hasFilters = !!status || !!search;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Ventas", to: "/sales" }, { label: "Registro de ventas" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <ShoppingCart className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Ventas
        </h1>
        <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => navigate("/sales/new")}>
          Nueva venta
        </Button>
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2 top-1/2 h-3 w-3 -translate-y-1/2 text-(--text-muted)" />
          <input
            type="search"
            placeholder="Cliente o número..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setOffset(0); }}
            className="rounded border border-(--border-color) bg-(--bg-primary) py-1 pl-6 pr-2 text-xs text-(--text-primary) transition-colors w-48"
          />
        </div>
        <select
          value={status}
          onChange={(e) => { setStatus(e.target.value as SaleStatus | ""); setOffset(0); }}
          className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1 text-xs text-(--text-primary) transition-colors"
        >
          <option value="">Todos los estados</option>
          {(Object.keys(STATUS_LABEL) as SaleStatus[]).map((s) => (
            <option key={s} value={s}>{STATUS_LABEL[s]}</option>
          ))}
        </select>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {sales === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : page.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">
          {hasFilters ? "No hay ventas que coincidan con los filtros." : "Todavía no has creado ninguna venta."}
        </p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Número</th>
                <th className="px-3 py-2 font-medium">Cliente</th>
                <th className="px-3 py-2 font-medium">Fecha</th>
                <th className="px-3 py-2 font-medium">Vence</th>
                <th className="px-3 py-2 font-medium">Total</th>
                <th className="px-3 py-2 font-medium">Estado</th>
              </tr>
            </thead>
            <tbody>
              {page.map((s, i) => {
                const total = s.lines.reduce((sum, l) => sum + l.total, 0);
                return (
                  <tr
                    key={s.id}
                    className={`cursor-pointer hover:bg-(--bg-hover) ${i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}`}
                    onClick={() => navigate(`/sales/${s.id}`)}
                  >
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{s.number || "Borrador"}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{customerName(s.customer_id)}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{new Date(s.issue_date).toLocaleDateString("es-CO")}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{s.due_date ? new Date(s.due_date).toLocaleDateString("es-CO") : "—"}</td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{formatCOP.format(total)}</td>
                    <td className="px-3 py-2"><StatusPill tone={STATUS_TONE[s.status]} label={STATUS_LABEL[s.status]} /></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {sales !== null && sales.length > 0 && (
        <Pagination
          offset={offset}
          count={page.length}
          hasNext={hasNext}
          loading={false}
          onPrev={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
          onNext={() => setOffset((o) => o + PAGE_SIZE)}
        />
      )}
    </div>
  );
}
