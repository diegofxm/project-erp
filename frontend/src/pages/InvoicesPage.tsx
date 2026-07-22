import { useEffect, useState } from "react";
import { FileText, Plus, Search, X } from "lucide-react";
import { useNavigate } from "react-router";
import { listDocuments } from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import type { Document, DocumentStatus } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Spinner } from "../components/ui/Spinner";
import { Pagination } from "../components/ui/Pagination";
import { StatusBadge } from "../components/invoice-form/StatusBadge";

const INVOICE_DIAN_DOCUMENT_TYPE = "01";
const PAGE_SIZE = 10;

const STATUS_OPTIONS: { value: DocumentStatus | ""; label: string }[] = [
  { value: "", label: "Todos los estados" },
  { value: "draft", label: "Borrador" },
  { value: "accepted", label: "Aceptado" },
  { value: "sent", label: "Enviado" },
  { value: "rejected", label: "Rechazado" },
  { value: "send_error", label: "Error de envío" },
];

export function InvoicesPage() {
  const [documents, setDocuments] = useState<Document[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const [loadingPage, setLoadingPage] = useState(false);
  const [status, setStatus] = useState<DocumentStatus | "">("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const navigate = useNavigate();

  useEffect(() => {
    const t = setTimeout(() => { setDebouncedSearch(search); setOffset(0); }, 400);
    return () => clearTimeout(t);
  }, [search]);

  const hasFilters = !!status || !!from || !!to || !!search;

  function resetFilters() {
    setStatus("");
    setFrom("");
    setTo("");
    setSearch("");
    setOffset(0);
  }

  useEffect(() => {
    setLoadingPage(true);
    listDocuments({
      dian_document_type_code: INVOICE_DIAN_DOCUMENT_TYPE,
      limit: PAGE_SIZE + 1,
      offset,
      ...(status && { status }),
      ...(from && { from }),
      ...(to && { to }),
      ...(debouncedSearch && { search: debouncedSearch }),
    })
      .then(setDocuments)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar las facturas"))
      .finally(() => setLoadingPage(false));
  }, [offset, status, from, to, debouncedSearch]);

  const hasNext = (documents?.length ?? 0) > PAGE_SIZE;
  const page = documents?.slice(0, PAGE_SIZE) ?? null;
  const hasRefs = page?.some((d) => (d.nc_count ?? 0) > 0 || (d.nd_count ?? 0) > 0) ?? false;

  return (
    <div className="p-4">
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <FileText className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Factura Electrónica
        </h1>
        <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => navigate("/documents/invoices/new")}>
          Nueva factura
        </Button>
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2 top-1/2 h-3 w-3 -translate-y-1/2 text-(--text-muted)" />
          <input
            type="search"
            placeholder="Cliente o número..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="rounded border border-(--border-color) bg-(--bg-primary) py-1 pl-6 pr-2 text-xs text-(--text-primary) transition-colors w-44"
          />
        </div>
        <select
          value={status}
          onChange={(e) => { setStatus(e.target.value as DocumentStatus | ""); setOffset(0); }}
          className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1 text-xs text-(--text-primary) transition-colors"
        >
          {STATUS_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
        <input
          type="date"
          value={from}
          onChange={(e) => { setFrom(e.target.value); setOffset(0); }}
          title="Desde"
          className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1 text-xs text-(--text-primary) transition-colors"
        />
        <input
          type="date"
          value={to}
          onChange={(e) => { setTo(e.target.value); setOffset(0); }}
          title="Hasta"
          className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1 text-xs text-(--text-primary) transition-colors"
        />
        {hasFilters && (
          <button type="button" onClick={resetFilters} className="flex items-center gap-1 text-xs text-(--text-muted) hover:text-(--text-primary) transition-colors">
            <X className="h-3 w-3" /> Limpiar
          </button>
        )}
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {page === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : page.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">
          {hasFilters ? "No hay facturas que coincidan con los filtros." : "Todavía no has creado ninguna factura."}
        </p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Número</th>
                <th className="px-3 py-2 font-medium">Cliente</th>
                <th className="px-3 py-2 font-medium">Total</th>
                <th className="px-3 py-2 font-medium">Estado</th>
                {hasRefs && <th className="px-3 py-2 font-medium">Referencias</th>}
                <th className="px-3 py-2 font-medium">Fecha</th>
              </tr>
            </thead>
            <tbody>
              {page.map((d, i) => (
                <tr
                  key={d.id}
                  className={`cursor-pointer hover:bg-(--bg-hover) ${i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}`}
                  onClick={() => navigate(`/documents/invoices/${d.id}`)}
                >
                  <td className="px-3 py-2 font-mono text-(--text-primary)">
                    {d.prefix && d.number ? `${d.prefix}${d.number}` : "Borrador"}
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">{d.customer.name}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(d.totals.payable_cents / 100)}</td>
                  <td className="px-3 py-2"><StatusBadge status={d.status} /></td>
                  {hasRefs && (
                    <td className="px-3 py-2">
                      <div className="flex items-center gap-1">
                        {(d.nc_count ?? 0) > 0 && (
                          <span translate="no" className="rounded px-1.5 py-0.5 text-[10px] font-medium bg-(--color-warning-bg) text-(--color-warning-text)">NC</span>
                        )}
                        {(d.nd_count ?? 0) > 0 && (
                          <span translate="no" className="rounded px-1.5 py-0.5 text-[10px] font-medium bg-(--color-info-bg) text-(--color-info-text)">ND</span>
                        )}
                      </div>
                    </td>
                  )}
                  <td className="px-3 py-2 text-(--text-secondary)">{new Date(d.created_at).toLocaleDateString("es-CO")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {page !== null && (
        <Pagination
          offset={offset}
          count={page.length}
          hasNext={hasNext}
          loading={loadingPage}
          onPrev={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
          onNext={() => setOffset((o) => o + PAGE_SIZE)}
        />
      )}
    </div>
  );
}
