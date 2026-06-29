import { useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { listDianDocumentTypes } from "../../lib/catalogs";
import { createNumberingRange, deactivateNumberingRange, listNumberingRanges } from "../../lib/numberingRanges";
import { ApiError } from "../../lib/apiClient";
import { useCatalog } from "../../lib/useCatalog";
import { useConfirm } from "../../context/ConfirmContext";
import { useToast } from "../../context/ToastContext";
import type { CreateNumberingRangePayload, NumberingRange, NumberingRangeStatus } from "../../lib/types";
import { Button } from "../ui/Button";
import { Banner } from "../ui/Banner";
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

  // "Eliminar" desactiva el rango (is_active=false) — nunca un borrado real, ver
  // numbering.Repository.Deactivate en apidian: un rango ya usado tiene FK real desde
  // documents.Document, y una resolución de numeración es un dato legal/auditable.
  async function handleDeactivate(r: NumberingRange) {
    if (!(await confirm(`¿Eliminar el rango "${r.prefix}"? Dejará de poder usarse para emitir facturas.`, { tone: "danger" }))) return;
    try {
      await deactivateNumberingRange(r.id);
      toast.success("Rango de numeración eliminado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el rango de numeración");
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded border border-(--border-color) p-4">
      <h2 className="text-xs font-semibold text-(--text-primary)">Rangos de numeración</h2>
      {loadError && <Banner tone="danger">{loadError}</Banner>}

      {ranges === null && (
        <div className="flex min-h-20 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      )}

      {ranges && ranges.length > 0 && (
        <div className="flex flex-col gap-1">
          {ranges.map((r) => (
            <div
              key={r.id}
              className="flex items-center justify-between rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 text-xs"
            >
              <span className="font-medium text-(--text-primary)">{docTypeName(r.dian_document_type_code)}</span>
              <span className="font-mono text-(--text-secondary)">
                {r.prefix} {r.range_from}–{r.range_to ?? "∞"}
              </span>
              <span className="text-(--text-muted)">Actual: {r.current_number}</span>
              <span className="text-(--text-muted)">Vence: {r.valid_to}</span>
              <span className="text-(--text-muted)">{r.environment === "1" ? "Producción" : "Habilitación"}</span>
              <span className={`rounded px-2 py-0.5 font-medium ${STATUS_CLASSES[r.status]}`}>{STATUS_LABELS[r.status]}</span>
              <button
                type="button"
                title="Eliminar"
                disabled={r.status === "inactive"}
                onClick={() => handleDeactivate(r)}
                className="rounded p-1 text-(--color-danger) hover:bg-(--bg-hover) disabled:cursor-not-allowed disabled:opacity-40"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}
      {ranges && ranges.length === 0 && !showForm && (
        <p className="text-xs text-(--text-secondary)">Todavía no hay rangos de numeración registrados.</p>
      )}

      {ranges !== null &&
        (showForm ? (
          <NumberingRangeForm onSubmit={handleCreate} onCancel={() => setShowForm(false)} loading={loading} />
        ) : (
          <Button
            type="button"
            variant="secondary"
            icon={<Plus className="h-3.5 w-3.5" />}
            onClick={() => setShowForm(true)}
            className="self-start"
          >
            Nuevo rango de numeración
          </Button>
        ))}
    </div>
  );
}
