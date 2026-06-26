import { useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { listDianDocumentTypes } from "../../lib/catalogs";
import { createNumberingRange, listNumberingRanges } from "../../lib/numberingRanges";
import { ApiError } from "../../lib/apiClient";
import { useCatalog } from "../../lib/useCatalog";
import type { CreateNumberingRangePayload, NumberingRange } from "../../lib/types";
import { Button } from "../ui/Button";
import { Banner } from "../ui/Banner";
import { Spinner } from "../ui/Spinner";
import { NumberingRangeForm } from "./NumberingRangeForm";

// Todo a todo el ancho disponible (ver docs/frontend-architecture.md, regla de ancho) —
// NumberingRangeForm redistribuye sus propios campos en una grilla auto-fit.
export function NumberingRangesPanel() {
  const [ranges, setRanges] = useState<NumberingRange[] | null>(null);
  const { data: docTypes } = useCatalog(listDianDocumentTypes);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function refresh() {
    listNumberingRanges()
      .then(setRanges)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar los rangos de numeración"));
  }

  useEffect(() => {
    refresh();
  }, []);

  function docTypeName(code: string): string {
    return docTypes.find((t) => t.code === code)?.name ?? code;
  }

  async function handleCreate(payload: CreateNumberingRangePayload) {
    setError(null);
    setLoading(true);
    try {
      await createNumberingRange(payload);
      setShowForm(false);
      refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo registrar el rango de numeración");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded border border-(--border-color) p-4">
      <h2 className="text-xs font-semibold text-(--text-primary)">Rangos de numeración</h2>
      {error && <Banner tone="danger">{error}</Banner>}

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
              <span className="text-(--text-muted)">{r.environment === "1" ? "Producción" : "Habilitación"}</span>
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
