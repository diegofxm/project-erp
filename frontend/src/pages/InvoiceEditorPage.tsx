import { useEffect, useState } from "react";
import { Send, Trash2 } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import { confirmDocument, createInvoiceDraft, deleteDraft, getDocument, updateInvoiceDraft } from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { useAuth } from "../context/AuthContext";
import type { Document, IssueInvoicePayload } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { InvoiceForm } from "../components/invoice-form/InvoiceForm";
import { StatusBadge } from "../components/invoice-form/StatusBadge";

// :id es "new" (crear) o un UUID real (editar mientras siga en draft, ver solo lectura en
// cualquier otro estado). Primer uso de un parámetro de ruta dinámico en este frontend.
export function InvoiceEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { activeIssuer } = useAuth();
  const isNew = id === "new";

  const [doc, setDoc] = useState<Document | null>(null);
  const [loadingDocument, setLoadingDocument] = useState(!isNew);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [confirming, setConfirming] = useState(false);

  useEffect(() => {
    if (isNew || !id) return;
    setLoadingDocument(true);
    getDocument(id)
      .then(setDoc)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la factura"))
      .finally(() => setLoadingDocument(false));
  }, [id, isNew]);

  async function handleSubmit(payload: IssueInvoicePayload) {
    if (!id) return;
    setError(null);
    setSaving(true);
    try {
      const saved = isNew ? await createInvoiceDraft(payload) : await updateInvoiceDraft(id, payload);
      if (isNew) {
        navigate(`/documents/invoices/${saved.id}`, { replace: true });
      } else {
        setDoc(saved);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar la factura");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!id || isNew) return;
    if (!window.confirm("¿Eliminar este borrador? Esta acción no se puede deshacer.")) return;
    try {
      await deleteDraft(id);
      navigate("/documents/invoices");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo eliminar el borrador");
    }
  }

  // confirmDocument reclama el consecutivo real, firma y —si el ambiente lo permite— envía a
  // la DIAN (ver docs/apidian-architecture.md sección 9.25). Único punto donde se "gasta" un
  // número real — antes de esto el borrador se podía editar o eliminar libremente.
  async function handleConfirm() {
    if (!id || isNew) return;
    if (!window.confirm("¿Confirmar y enviar esta factura? Reclama el número real ante la DIAN y ya no se puede editar ni eliminar.")) return;
    setError(null);
    setConfirming(true);
    try {
      setDoc(await confirmDocument(id));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo confirmar la factura");
    } finally {
      setConfirming(false);
    }
  }

  const issuerNotReady = !activeIssuer?.has_software_credentials || !activeIssuer?.has_certificate;

  if (loadingDocument) {
    return (
      <div className="flex min-h-32 items-center justify-center p-4">
        <Spinner className="h-5 w-5 text-(--text-muted)" />
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="mb-3 flex items-center justify-between">
        <h1 className="text-sm font-semibold text-(--text-primary)">
          {isNew ? "Nueva factura" : doc ? `Factura ${doc.prefix ?? ""}${doc.number ?? "(borrador)"}` : "Factura"}
        </h1>
        {doc && <StatusBadge status={doc.status} />}
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {!isNew && doc?.status === "draft" && issuerNotReady && (
        <Banner tone="info">
          La empresa todavía no tiene software/certificado configurados — complétalos en Configuración → Empresa antes de confirmar.
        </Banner>
      )}

      {isNew || doc?.status === "draft" ? (
        <>
          <Card className="mt-3">
            <InvoiceForm initial={isNew ? null : doc} onSubmit={handleSubmit} onCancel={() => navigate("/documents/invoices")} loading={saving} />
          </Card>
          {!isNew && doc?.status === "draft" && (
            <div className="mt-3 flex gap-2">
              <Button
                type="button"
                variant="danger"
                icon={<Trash2 className="h-3.5 w-3.5" />}
                onClick={handleDelete}
                disabled={confirming}
              >
                Eliminar borrador
              </Button>
              <Button
                type="button"
                variant="success"
                icon={<Send className="h-3.5 w-3.5" />}
                onClick={handleConfirm}
                disabled={issuerNotReady}
                loading={confirming}
              >
                Confirmar y enviar
              </Button>
            </div>
          )}
        </>
      ) : doc ? (
        <Card className="mt-3 flex flex-col gap-4 p-4">
          <div className="grid grid-cols-12 gap-3 text-xs">
            <div className="col-span-4">
              <span className="text-(--text-secondary)">Cliente</span>
              <p className="text-(--text-primary)">{doc.customer.name}</p>
            </div>
            <div className="col-span-4">
              <span className="text-(--text-secondary)">Total</span>
              <p className="font-mono text-(--text-primary)">{formatCOP.format(doc.totals.payable_cents / 100)}</p>
            </div>
            <div className="col-span-4">
              <span className="text-(--text-secondary)">CUFE</span>
              <p className="break-all font-mono text-(--text-primary)">{doc.document_key || "—"}</p>
            </div>
          </div>

          {doc.qr_url && (
            <p className="text-xs">
              <a href={doc.qr_url} target="_blank" rel="noreferrer" className="text-(--accent-primary) hover:underline">
                Ver representación gráfica en el portal de la DIAN
              </a>
            </p>
          )}

          {(doc.dian_status_code || doc.dian_status_description) && (
            <div className="rounded border border-(--border-color) bg-(--bg-primary) p-3 text-xs">
              <p className="font-medium text-(--text-primary)">Estado DIAN: {doc.dian_status_code ?? "—"}</p>
              {doc.dian_status_description && <p className="mt-1 text-(--text-secondary)">{doc.dian_status_description}</p>}
              {doc.dian_status_message && <p className="mt-1 text-(--text-secondary)">{doc.dian_status_message}</p>}
            </div>
          )}

          <div className="overflow-hidden rounded border border-(--border-color)">
            <table className="w-full text-left text-xs">
              <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                <tr>
                  <th className="px-3 py-2 font-medium">Descripción</th>
                  <th className="px-3 py-2 font-medium">Cant.</th>
                  <th className="px-3 py-2 font-medium">Precio unitario</th>
                  <th className="px-3 py-2 font-medium">Total línea</th>
                </tr>
              </thead>
              <tbody>
                {doc.lines.map((line, i) => (
                  <tr key={i} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                    <td className="px-3 py-2 text-(--text-primary)">{line.description}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{line.quantity}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(line.unit_price_cents / 100)}</td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">
                      {formatCOP.format((line.line_extension_cents + (line.taxes?.reduce((s, t) => s + t.tax_amount_cents, 0) ?? 0)) / 100)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      ) : null}
    </div>
  );
}
