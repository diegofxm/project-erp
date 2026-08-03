import { useEffect, useState } from "react";
import { BookText, Plus } from "lucide-react";
import { useNavigate } from "react-router";
import { listJournals } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import type { JournalEntry, JournalStatus } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Spinner } from "../components/ui/Spinner";
import { Pagination } from "../components/ui/Pagination";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { InfoTip } from "../components/ui/InfoTip";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

const PAGE_SIZE = 20;

const STATUS_LABEL: Record<JournalStatus, string> = { DRAFT: "Borrador", POSTED: "Contabilizado", VOID: "Anulado" };
const STATUS_TONE: Record<JournalStatus, StatusTone> = { DRAFT: "neutral", POSTED: "success", VOID: "danger" };

function lineTotals(entry: JournalEntry) {
  return entry.lines.reduce(
    (acc, l) => ({ debit: acc.debit + l.debit, credit: acc.credit + l.credit }),
    { debit: 0, credit: 0 },
  );
}

export function AccountingJournalsPage() {
  const [entries, setEntries] = useState<JournalEntry[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  function refresh(off: number) {
    setLoading(true);
    listJournals(PAGE_SIZE, off)
      .then(setEntries)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar los asientos"))
      .finally(() => setLoading(false));
  }

  useEffect(() => { refresh(offset); }, [offset]);

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Asientos" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <BookText className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Asientos contables
          <InfoTip>
            Historial de todos los asientos: los <strong>automáticos</strong> se generan al confirmar una venta,
            recibir una compra o generar nómina; los <strong>manuales</strong> los registras tú (ej. ajustes,
            causaciones). Un asiento anulado no borra el registro — queda marcado como reversado.
          </InfoTip>
        </h1>
        <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => navigate("/accounting/journals/new")}>
          Nuevo asiento
        </Button>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {entries === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : entries.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Todavía no hay asientos registrados.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Fecha</th>
                <th className="px-3 py-2 font-medium">Descripción</th>
                <th className="px-3 py-2 font-medium">Comprobante</th>
                <th className="px-3 py-2 font-medium">Origen</th>
                <th className="px-3 py-2 font-medium">Débito</th>
                <th className="px-3 py-2 font-medium">Crédito</th>
                <th className="px-3 py-2 font-medium">Estado</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e, i) => {
                const totals = lineTotals(e);
                return (
                  <tr
                    key={e.id}
                    className={`cursor-pointer hover:bg-(--bg-hover) ${i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}`}
                    onClick={() => navigate(`/accounting/journals/${e.id}`)}
                  >
                    <td className="px-3 py-2 text-(--text-secondary)">{new Date(e.date).toLocaleDateString("es-CO")}</td>
                    <td className="px-3 py-2 text-(--text-primary)">{e.description}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{e.voucher_number || "—"}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{e.source}</td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{formatCOP.format(totals.debit / 100)}</td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{formatCOP.format(totals.credit / 100)}</td>
                    <td className="px-3 py-2"><StatusPill tone={STATUS_TONE[e.status]} label={STATUS_LABEL[e.status]} /></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {entries !== null && entries.length > 0 && (
        <Pagination
          offset={offset}
          count={entries.length}
          hasNext={entries.length === PAGE_SIZE}
          loading={loading}
          onPrev={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
          onNext={() => setOffset((o) => o + PAGE_SIZE)}
        />
      )}
    </div>
  );
}
