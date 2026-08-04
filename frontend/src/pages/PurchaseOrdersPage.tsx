import { useEffect, useMemo, useState } from "react";
import { ShoppingBag, Plus, Search } from "lucide-react";
import { useNavigate } from "react-router";
import { listPurchases } from "../lib/purchases";
import { listSuppliers } from "../lib/suppliers";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import type { Purchase, PurchaseStatus, Supplier } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Spinner } from "../components/ui/Spinner";
import { Pagination } from "../components/ui/Pagination";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

const PAGE_SIZE = 10;

const STATUS_LABEL: Record<PurchaseStatus, string> = {
  draft: "Borrador", confirmed: "Confirmada", received: "Recibida", cancelled: "Cancelada",
};
const STATUS_TONE: Record<PurchaseStatus, StatusTone> = {
  draft: "neutral", confirmed: "info", received: "success", cancelled: "danger",
};

export function PurchaseOrdersPage() {
  const [purchases, setPurchases] = useState<Purchase[] | null>(null);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<PurchaseStatus | "">("");
  const [search, setSearch] = useState("");
  const [offset, setOffset] = useState(0);
  const navigate = useNavigate();

  useEffect(() => {
    listPurchases()
      .then(setPurchases)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar las órdenes de compra"));
    listSuppliers().then(setSuppliers).catch(() => setSuppliers([]));
  }, []);

  const supplierName = useMemo(() => {
    const map = new Map(suppliers.map((s) => [s.id, s.name]));
    return (id: string) => map.get(id) ?? "—";
  }, [suppliers]);

  const filtered = useMemo(() => {
    if (!purchases) return [];
    return purchases.filter((o) => {
      if (status && o.status !== status) return false;
      if (search.trim()) {
        const q = search.toLowerCase();
        const matchesNumber = o.number?.toLowerCase().includes(q);
        const matchesSupplier = supplierName(o.supplier_id).toLowerCase().includes(q);
        if (!matchesNumber && !matchesSupplier) return false;
      }
      return true;
    });
  }, [purchases, status, search, supplierName]);

  const page = filtered.slice(offset, offset + PAGE_SIZE);
  const hasNext = offset + PAGE_SIZE < filtered.length;
  const hasFilters = !!status || !!search;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Compras", to: "/purchases" }, { label: "Registro de compras" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <ShoppingBag className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Compras
        </h1>
        <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => navigate("/purchases/new")}>
          Nueva orden de compra
        </Button>
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2 top-1/2 h-3 w-3 -translate-y-1/2 text-(--text-muted)" />
          <input
            type="search"
            placeholder="Proveedor o número..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setOffset(0); }}
            className="rounded border border-(--border-color) bg-(--bg-primary) py-1 pl-6 pr-2 text-xs text-(--text-primary) transition-colors w-48"
          />
        </div>
        <select
          value={status}
          onChange={(e) => { setStatus(e.target.value as PurchaseStatus | ""); setOffset(0); }}
          className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1 text-xs text-(--text-primary) transition-colors"
        >
          <option value="">Todos los estados</option>
          {(Object.keys(STATUS_LABEL) as PurchaseStatus[]).map((s) => (
            <option key={s} value={s}>{STATUS_LABEL[s]}</option>
          ))}
        </select>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {purchases === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : page.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">
          {hasFilters ? "No hay órdenes que coincidan con los filtros." : "Todavía no has creado ninguna orden de compra."}
        </p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Número</th>
                <th className="px-3 py-2 font-medium">Proveedor</th>
                <th className="px-3 py-2 font-medium">Fecha</th>
                <th className="px-3 py-2 font-medium">Recepción esperada</th>
                <th className="px-3 py-2 font-medium">Total</th>
                <th className="px-3 py-2 font-medium">Estado</th>
              </tr>
            </thead>
            <tbody>
              {page.map((o, i) => {
                const total = o.lines.reduce((sum, l) => sum + l.total, 0);
                return (
                  <tr
                    key={o.id}
                    className={`cursor-pointer hover:bg-(--bg-hover) ${i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}`}
                    onClick={() => navigate(`/purchases/${o.id}`)}
                  >
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{o.number || "Borrador"}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{supplierName(o.supplier_id)}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{new Date(o.issue_date).toLocaleDateString("es-CO")}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{o.due_date ? new Date(o.due_date).toLocaleDateString("es-CO") : "—"}</td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{formatCOP.format(total)}</td>
                    <td className="px-3 py-2"><StatusPill tone={STATUS_TONE[o.status]} label={STATUS_LABEL[o.status]} /></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {purchases !== null && purchases.length > 0 && (
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
