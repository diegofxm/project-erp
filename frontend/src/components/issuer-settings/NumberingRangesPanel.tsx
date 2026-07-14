import { useEffect, useState } from "react";
import { Archive, Ban, Plus, RotateCcw } from "lucide-react";
import { Pagination } from "../ui/Pagination";
import { listDianDocumentTypes } from "../../lib/catalogs";
import { activateNumberingRange, createNumberingRange, deactivateNumberingRange, listNumberingRanges } from "../../lib/numberingRanges";
import { ApiError } from "../../lib/apiClient";
import { useCatalog } from "../../lib/useCatalog";
import { useConfirm } from "../../context/ConfirmContext";
import { useToast } from "../../context/ToastContext";
import type { CreateNumberingRangePayload, NumberingRange, NumberingRangeStatus } from "../../lib/types";
import { Button } from "../ui/Button";
import { Banner } from "../ui/Banner";
import { Card } from "../ui/Card";
import { Spinner } from "../ui/Spinner";
import { NumberingRangeForm } from "./NumberingRangeForm";

// Mismos tokens pastel que StatusBadge.tsx (sección 2.3 del design system) — "exhausted" usa
// el tono info (no es un error, una resolución agotada es esperable), "inactive" el tono
// neutro que ya usa StatusBadge para draft/built.
const STATUS_LABELS: Record<NumberingRangeStatus, string> = {
  active: "Activo",
  expired: "Vencido",
  exhausted: "Agotado",
  inactive: "Inactivo",
};
const STATUS_CLASSES: Record<NumberingRangeStatus, string> = {
  active: "bg-(--color-success-bg) text-(--color-success-text)",
  expired: "bg-(--color-danger-bg) text-(--color-danger-text)",
  exhausted: "bg-(--color-info-bg) text-(--color-info-text)",
  inactive: "bg-(--bg-tertiary) text-(--text-secondary)",
};

const RANGES_PAGE_SIZE = 5;

// Un rango inactivo "vale la pena reactivar" si su fecha de vencimiento es futura Y no está
// agotado — de lo contrario, reactivarlo lo pondría inmediatamente en estado expired/exhausted.
function wouldBeUsable(r: NumberingRange): boolean {
  if (new Date(r.valid_to) < new Date()) return false;
  if (r.range_to !== undefined && r.current_number >= r.range_to) return false;
  return true;
}

function RangesTable({ rows, docTypeName, onDeactivate, onArchive, onActivate }: {
  rows: NumberingRange[];
  docTypeName: (code: string) => string;
  onDeactivate: (r: NumberingRange) => void;
  onArchive: (r: NumberingRange) => void;
  onActivate: (r: NumberingRange) => void;
}) {
  const [offset, setOffset] = useState(0);
  const page = rows.slice(offset, offset + RANGES_PAGE_SIZE);
  const hasNext = offset + RANGES_PAGE_SIZE < rows.length;

  return (
    <>
      <div className="overflow-hidden rounded border border-(--border-color)">
        <table className="w-full text-left text-xs">
          <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
            <tr>
              <th className="px-3 py-2 font-medium">Tipo</th>
              <th className="px-3 py-2 font-medium">Resolución</th>
              <th className="px-3 py-2 font-medium">Actual</th>
              <th className="px-3 py-2 font-medium">Vence</th>
              <th className="px-3 py-2 font-medium">Ambiente</th>
              <th className="px-3 py-2 font-medium">Estado</th>
              <th className="px-3 py-2" />
            </tr>
          </thead>
          <tbody>
            {page.map((r, i) => (
              <tr key={r.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                <td className="whitespace-nowrap px-3 py-2 font-medium text-(--text-primary)">{docTypeName(r.dian_document_type_code)}</td>
                <td className="whitespace-nowrap px-3 py-2 font-mono text-(--text-secondary)">{r.prefix} {r.range_from}–{r.range_to ?? "∞"}</td>
                <td className="whitespace-nowrap px-3 py-2 text-(--text-muted)">{r.current_number}</td>
                <td className="whitespace-nowrap px-3 py-2 text-(--text-muted)">{r.valid_to}</td>
                <td className="whitespace-nowrap px-3 py-2 text-(--text-muted)">{r.environment === "1" ? "Producción" : "Habilitación"}</td>
                <td className="whitespace-nowrap px-3 py-2">
                  <span className={`rounded px-2 py-0.5 font-medium ${STATUS_CLASSES[r.status]}`}>{STATUS_LABELS[r.status]}</span>
                </td>
                <td className="px-3 py-2 text-right">
                  <div className="flex items-center justify-end gap-1">
                    {r.status === "active" && (
                      <button
                        type="button"
                        title="Desactivar"
                        onClick={() => onDeactivate(r)}
                        className="rounded p-1 text-(--color-danger) transition-colors hover:bg-(--bg-hover)"
                      >
                        <Ban className="h-3.5 w-3.5" />
                      </button>
                    )}
                    {(r.status === "expired" || r.status === "exhausted") && (
                      <button
                        type="button"
                        title="Archivar"
                        onClick={() => onArchive(r)}
                        className="rounded p-1 text-(--text-muted) transition-colors hover:bg-(--bg-hover)"
                      >
                        <Archive className="h-3.5 w-3.5" />
                      </button>
                    )}
                    {r.status === "inactive" && wouldBeUsable(r) && (
                      <button
                        type="button"
                        title="Reactivar"
                        onClick={() => onActivate(r)}
                        className="rounded p-1 text-(--color-success) transition-colors hover:bg-(--bg-hover)"
                      >
                        <RotateCcw className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Pagination
        offset={offset}
        count={page.length}
        hasNext={hasNext}
        onPrev={() => setOffset((o) => Math.max(0, o - RANGES_PAGE_SIZE))}
        onNext={() => setOffset((o) => o + RANGES_PAGE_SIZE)}
      />
    </>
  );
}

// Todo a todo el ancho disponible (ver docs/frontend-architecture.md, regla de ancho) —
// NumberingRangeForm redistribuye sus propios campos en una grilla auto-fit.
export function NumberingRangesPanel() {
  const [ranges, setRanges] = useState<NumberingRange[] | null>(null);
  const { data: docTypes } = useCatalog(listDianDocumentTypes);
  const [showForm, setShowForm] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const confirm = useConfirm();
  const toast = useToast();

  function refresh() {
    listNumberingRanges()
      .then(setRanges)
      .catch((err) => setLoadError(err instanceof ApiError ? err.message : "No se pudieron cargar los rangos de numeración"));
  }

  useEffect(() => {
    refresh();
  }, []);

  function docTypeName(code: string): string {
    return docTypes.find((t) => t.code === code)?.name ?? code;
  }

  async function handleCreate(payload: CreateNumberingRangePayload) {
    setLoading(true);
    try {
      await createNumberingRange(payload);
      setShowForm(false);
      toast.success("Rango de numeración registrado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo registrar el rango de numeración");
    } finally {
      setLoading(false);
    }
  }

  async function handleDeactivate(r: NumberingRange) {
    if (!(await confirm(`¿Desactivar el rango "${r.prefix}"? Dejará de usarse para emitir facturas — podrás reactivarlo después si sigue siendo válido.`, { tone: "danger" }))) return;
    try {
      await deactivateNumberingRange(r.id);
      toast.success("Rango desactivado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo desactivar el rango");
    }
  }

  // Archivar = mismo endpoint que desactivar, pero para rangos vencidos o agotados que el
  // usuario quiere ocultar de la vista activa sin reactivarlos (semánticamente es "ya no me
  // importa ver esto, pero no lo borro porque es un registro legal").
  async function handleArchive(r: NumberingRange) {
    if (!(await confirm(`¿Archivar el rango "${r.prefix}"? Quedará oculto en esta vista. No se puede reactivar porque ya está ${r.status === "expired" ? "vencido" : "agotado"}.`))) return;
    try {
      await deactivateNumberingRange(r.id);
      toast.success("Rango archivado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo archivar el rango");
    }
  }

  async function handleActivate(r: NumberingRange) {
    if (!(await confirm(`¿Reactivar el rango "${r.prefix}"? Volverá a estar disponible para emitir facturas.`))) return;
    try {
      await activateNumberingRange(r.id);
      toast.success("Rango reactivado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo reactivar el rango");
    }
  }

  const activeRanges = ranges?.filter((r) => r.status === "active") ?? [];
  const inactiveRanges = ranges?.filter((r) => r.status !== "active") ?? [];

  return (
    <Card className="flex flex-col gap-3 p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xs font-semibold text-(--text-primary)">Rangos de numeración</h2>
        {ranges !== null && !showForm && (
          <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowForm(true)}>
            Nuevo rango
          </Button>
        )}
      </div>

      {loadError && <Banner tone="danger">{loadError}</Banner>}

      {ranges === null && (
        <div className="flex min-h-20 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      )}

      {showForm && (
        <NumberingRangeForm onSubmit={handleCreate} onCancel={() => setShowForm(false)} loading={loading} />
      )}

      {ranges?.length === 0 && !showForm && (
        <p className="text-xs text-(--text-secondary)">Todavía no hay rangos de numeración registrados.</p>
      )}

      {activeRanges.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-medium text-(--text-secondary)">Activos</p>
          <RangesTable rows={activeRanges} docTypeName={docTypeName} onDeactivate={handleDeactivate} onArchive={handleArchive} onActivate={handleActivate} />
        </div>
      )}

      {activeRanges.length > 0 && inactiveRanges.length > 0 && (
        <div className="border-t border-(--border-light)" />
      )}

      {inactiveRanges.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-medium text-(--text-muted)">Inactivos y vencidos</p>
          <RangesTable rows={inactiveRanges} docTypeName={docTypeName} onDeactivate={handleDeactivate} onArchive={handleArchive} onActivate={handleActivate} />
        </div>
      )}
    </Card>
  );
}
