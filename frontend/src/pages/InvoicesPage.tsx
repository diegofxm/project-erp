import { useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { useNavigate } from "react-router";
import { listDocuments } from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import type { Document } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Spinner } from "../components/ui/Spinner";
import { StatusBadge } from "../components/invoice-form/StatusBadge";

const INVOICE_DIAN_DOCUMENT_TYPE = "01";

// Listado a todo el ancho (ver docs/frontend-architecture.md, regla de ancho) — mismo patrón
// que CustomersPage/ProductsPage, pero sin formulario inline: crear/editar/ver una factura
// necesita su propia ruta (InvoiceEditorPage), dado el tamaño del formulario.
export function InvoicesPage() {
  const [documents, setDocuments] = useState<Document[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    listDocuments({ dian_document_type_code: INVOICE_DIAN_DOCUMENT_TYPE })
      .then(setDocuments)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar las facturas"));
  }, []);

  return (
    <div className="p-4">
      <div className="mb-3 flex items-center justify-between">
        <h1 className="text-sm font-semibold text-(--text-primary)">Factura Electrónica</h1>
        <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => navigate("/documents/invoices/new")}>
          Nueva factura
        </Button>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {documents === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : documents.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Todavía no has creado ninguna factura.</p>
      ) : (
        <div className="overflow-hidden rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Número</th>
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
                  onClick={() => navigate(`/documents/invoices/${d.id}`)}
                >
                  <td className="px-3 py-2 font-mono text-(--text-primary)">
                    {d.prefix && d.number ? `${d.prefix}${d.number}` : "Borrador"}
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">{d.customer.name}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(d.totals.payable_cents / 100)}</td>
                  <td className="px-3 py-2">
                    <StatusBadge status={d.status} />
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">{new Date(d.created_at).toLocaleDateString("es-CO")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
