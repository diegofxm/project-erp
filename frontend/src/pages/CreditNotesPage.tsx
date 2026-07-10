import { useEffect, useState } from "react";
import { ChevronLeft, ChevronRight, FileMinus } from "lucide-react";
import { useNavigate } from "react-router";
import { listDocuments } from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import type { Document } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Spinner } from "../components/ui/Spinner";
import { StatusBadge } from "../components/invoice-form/StatusBadge";

const CREDIT_NOTE_DIAN_DOCUMENT_TYPE = "91";
const PAGE_SIZE = 50;

export function CreditNotesPage() {
  const [documents, setDocuments] = useState<Document[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const [loadingPage, setLoadingPage] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    setLoadingPage(true);
    listDocuments({ dian_document_type_code: CREDIT_NOTE_DIAN_DOCUMENT_TYPE, limit: PAGE_SIZE, offset })
      .then(setDocuments)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar las notas crédito"))
      .finally(() => setLoadingPage(false));
  }, [offset]);

  const hasPrevious = offset > 0;
  const hasNext = (documents?.length ?? 0) === PAGE_SIZE;

  return (
    <div className="p-4">
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
            <FileMinus className="h-4 w-4 shrink-0 text-(--text-secondary)" />
            Nota Crédito
          </h1>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {documents === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : documents.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">
          Todavía no has emitido ninguna nota crédito. Para crear una, abre una factura confirmada y usa el botón
          "Emitir Nota Crédito".
        </p>
      ) : (
        <div className="overflow-hidden rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Número</th>
                <th className="px-3 py-2 font-medium">Factura referenciada</th>
                <th className="px-3 py-2 font-medium">Cliente</th>
                <th className="px-3 py-2 font-medium">Total</th>
                <th className="px-3 py-2 font-medium">Estado</th>
                <th className="px-3 py-2 font-medium">Fecha</th>
              </tr>
            </thead>
            <tbody>
              {documents.map((d, i) => (
                <tr
                  key={d.id}
                  className={`cursor-pointer hover:bg-(--bg-hover) ${i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}`}
                  onClick={() => navigate(`/documents/credit-notes/${d.id}`)}
                >
                  <td className="px-3 py-2 font-mono text-(--text-primary)">
                    {d.prefix && d.number ? `${d.prefix}${d.number}` : "Borrador"}
                  </td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">
                    {d.billing_reference
                      ? `${d.billing_reference.prefix}${d.billing_reference.number}`
                      : "—"}
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">{d.customer.name}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">
                    {formatCOP.format(d.totals.payable_cents / 100)}
                  </td>
                  <td className="px-3 py-2">
                    <StatusBadge status={d.status} />
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">
                    {new Date(d.created_at).toLocaleDateString("es-CO")}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {documents !== null && (hasPrevious || hasNext) && (
        <div className="mt-3 flex items-center justify-between">
          <span className="text-xs text-(--text-secondary)">
            Mostrando {offset + 1}–{offset + documents.length}
          </span>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="secondary"
              icon={<ChevronLeft className="h-3.5 w-3.5" />}
              disabled={!hasPrevious || loadingPage}
              onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
            >
              Anterior
            </Button>
            <Button
              type="button"
              variant="secondary"
              icon={<ChevronRight className="h-3.5 w-3.5" />}
              disabled={!hasNext || loadingPage}
              loading={loadingPage}
              onClick={() => setOffset((o) => o + PAGE_SIZE)}
            >
              Siguiente
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
