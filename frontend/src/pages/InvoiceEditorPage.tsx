import { useEffect, useState } from "react";
import { ExternalLink, FileCode, FileMinus, FilePlus, FileText, Mail, Send, Trash2 } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import {
  confirmDocument,
  createInvoiceDraft,
  deleteDraft,
  getDocument,
  getDocumentPdfBlobUrl,
  getDocumentXmlBlobUrl,
  sendDocumentEmail,
  updateInvoiceDraft,
} from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { openInNewTab } from "../lib/openInNewTab";
import { formatCOP } from "../lib/currency";
import { idTypeLabel } from "../lib/idTypes";
import { useAuth } from "../context/AuthContext";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import { usePdfFormat } from "../lib/usePdfFormat";
import type { Document, IssueInvoicePayload } from "../lib/types";
import { BackLink } from "../components/ui/BackLink";
import { Banner } from "../components/ui/Banner";
import { DianStatusBlock } from "../components/DianStatusBlock";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { InvoiceForm } from "../components/invoice-form/InvoiceForm";
import { StatusBadge } from "../components/invoice-form/StatusBadge";

const NOTE_TYPE_LABEL: Record<string, string> = {
  "91": "Nota Crédito",
  "92": "Nota Débito",
  "95": "Nota de Ajuste",
};

// :id es "new" (crear) o un UUID real (editar mientras siga en draft, ver solo lectura en
// cualquier otro estado). Primer uso de un parámetro de ruta dinámico en este frontend.
export function InvoiceEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { activeIssuer } = useAuth();
  const confirmDialog = useConfirm();
  const toast = useToast();
  const isNew = id === "new";

  const [doc, setDoc] = useState<Document | null>(null);
  const [loadingDocument, setLoadingDocument] = useState(!isNew);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [loadingPdf, setLoadingPdf] = useState(false);
  const [loadingXml, setLoadingXml] = useState(false);
  const [sendingEmail, setSendingEmail] = useState(false);
  const [pdfFormat] = usePdfFormat();

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
    if (!(await confirmDialog("¿Eliminar este borrador? Esta acción no se puede deshacer.", { tone: "danger" }))) return;
    try {
      await deleteDraft(id);
      toast.success("Borrador eliminado.");
      navigate("/documents/invoices");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el borrador");
    }
  }

  // confirmDocument reclama el consecutivo real, firma y —si el ambiente lo permite— envía a
  // la DIAN (ver docs/apidian-architecture.md sección 9.25). Único punto donde se "gasta" un
  // número real — antes de esto el borrador se podía editar o eliminar libremente.
  async function handleConfirm() {
    if (!id || isNew) return;
    if (
      !(await confirmDialog("¿Confirmar y enviar esta factura? Reclama el número real ante la DIAN y ya no se puede editar ni eliminar.", {
        confirmLabel: "Confirmar y enviar",
      }))
    )
      return;
    setConfirming(true);
    try {
      setDoc(await confirmDocument(id));
      toast.success("Factura confirmada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo confirmar la factura");
    } finally {
      setConfirming(false);
    }
  }

  async function handleViewPdf() {
    if (!id || isNew) return;
    setLoadingPdf(true);
    try {
      const url = await getDocumentPdfBlobUrl(id, pdfFormat);
      openInNewTab(url);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo generar el PDF");
    } finally {
      setLoadingPdf(false);
    }
  }

  async function handleDownloadXml() {
    if (!id || isNew) return;
    setLoadingXml(true);
    try {
      const url = await getDocumentXmlBlobUrl(id);
      const a = document.createElement("a");
      a.href = url;
      a.download = doc ? `${doc.prefix ?? ""}${doc.number ?? ""}.xml` : "documento.xml";
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo descargar el XML");
    } finally {
      setLoadingXml(false);
    }
  }

  // handleSendEmail envía la Factura ya accepted al correo del cliente con el PDF y el XML
  // firmados adjuntos (ver docs/apidian-architecture.md sección 9.42) — visible para un
  // tercero real, por eso pide confirmación igual que eliminar/confirmar.
  async function handleSendEmail() {
    if (!id || isNew || !doc) return;
    if (!(await confirmDialog(`¿Enviar esta factura por correo a ${doc.customer.email || "el cliente"}?`))) return;
    setSendingEmail(true);
    try {
      await sendDocumentEmail(id, pdfFormat);
      toast.success(`Factura enviada a ${doc.customer.email}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo enviar la factura por correo");
    } finally {
      setSendingEmail(false);
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
      <BackLink to="/documents/invoices" label="Facturas Electrónicas" />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
            <FileText className="h-4 w-4 shrink-0 text-(--accent-primary)" />
            {isNew ? "Nueva factura" : doc ? `Factura ${doc.prefix ?? ""}${doc.number ?? "(borrador)"}` : "Factura"}
          </h1>
        <div className="flex flex-wrap items-center gap-1.5">
          {doc && <StatusBadge status={doc.status} />}
          {!isNew && doc && (
            <Button type="button" variant="secondary" icon={<FileText className="h-3.5 w-3.5" />} loading={loadingPdf} onClick={handleViewPdf}>
              Ver PDF
            </Button>
          )}
          {!isNew && doc && doc.status !== "draft" && (
            <Button type="button" variant="secondary" icon={<FileCode className="h-3.5 w-3.5" />} loading={loadingXml} onClick={handleDownloadXml}>
              Descargar XML
            </Button>
          )}
          {!isNew && doc?.status === "accepted" && (
            <Button type="button" variant="secondary" icon={<Mail className="h-3.5 w-3.5" />} loading={sendingEmail} onClick={handleSendEmail}>
              Enviar al cliente
            </Button>
          )}
          {!isNew && doc?.status === "accepted" && (() => {
            const draftNC = doc.related_notes?.find(n => n.dian_document_type_code === "91" && n.status === "draft");
            return (
              <Button
                type="button"
                variant="secondary"
                icon={<FileMinus className="h-3.5 w-3.5" />}
                onClick={() => navigate(draftNC ? `/documents/credit-notes/${draftNC.id}` : `/documents/credit-notes/new?from=${id}`)}
              >
                {draftNC ? "Editar borrador NC" : "Emitir Nota Crédito"}
              </Button>
            );
          })()}
          {!isNew && doc?.status === "accepted" && (() => {
            const draftND = doc.related_notes?.find(n => n.dian_document_type_code === "92" && n.status === "draft");
            return (
              <Button
                type="button"
                variant="secondary"
                icon={<FilePlus className="h-3.5 w-3.5" />}
                onClick={() => navigate(draftND ? `/documents/debit-notes/${draftND.id}` : `/documents/debit-notes/new?from=${id}`)}
              >
                {draftND ? "Editar borrador ND" : "Emitir Nota Débito"}
              </Button>
            );
          })()}
          {!isNew && doc?.status === "draft" && (
            <>
              <Button type="button" variant="danger" icon={<Trash2 className="h-3.5 w-3.5" />} onClick={handleDelete} disabled={confirming}>
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
            </>
          )}
        </div>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {!isNew && doc?.status === "draft" && issuerNotReady && (
        <Banner tone="info">
          La empresa todavía no tiene software/certificado configurados — complétalos en Configuración → Empresa antes de confirmar.
        </Banner>
      )}

      {isNew || doc?.status === "draft" ? (
        <Card className="mt-3">
          <InvoiceForm initial={isNew ? null : doc} onSubmit={handleSubmit} onCancel={() => navigate("/documents/invoices")} loading={saving} />
        </Card>
      ) : doc ? (
        <Card className="mt-3 flex flex-col gap-4 p-4">
          <div className="grid grid-cols-12 gap-3 text-xs">
            <div className="col-span-6 sm:col-span-3">
              <span className="text-(--text-secondary)">Cliente</span>
              <p className="text-(--text-primary)">{doc.customer.name}</p>
              <p className="text-(--text-muted)">{idTypeLabel(doc.customer.identification.type_code)} {doc.customer.identification.number}</p>
            </div>
            <div className="col-span-6 sm:col-span-3">
              <span className="text-(--text-secondary)">Total</span>
              <p className="font-mono text-(--text-primary)">{formatCOP.format(doc.totals.payable_cents / 100)}</p>
            </div>
            <div className="col-span-6 sm:col-span-3">
              <span className="text-(--text-secondary)">Impuestos</span>
              <p className="font-mono text-(--text-primary)">
                {formatCOP.format((doc.totals.tax_inclusive_cents - doc.totals.tax_exclusive_cents) / 100)}
              </p>
            </div>
            <div className="col-span-6 sm:col-span-3">
              <span className="text-(--text-secondary)">Fecha de emisión</span>
              <p className="text-(--text-primary)">{doc.issue_date ? new Date(doc.issue_date).toLocaleDateString("es-CO") : "—"}</p>
            </div>
            <div className="col-span-12">
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

          <DianStatusBlock statusCode={doc.dian_status_code} description={doc.dian_status_description} message={doc.dian_status_message} />

          {doc.related_notes && doc.related_notes.length > 0 && (
            <div>
              <p className="mb-1.5 text-xs font-semibold text-(--text-primary)">
                Notas emitidas ({doc.related_notes.length})
              </p>
              <div className="flex flex-col gap-1">
                {doc.related_notes.map((note) => {
                  const route = note.dian_document_type_code === "91"
                    ? `/documents/credit-notes/${note.id}`
                    : `/documents/debit-notes/${note.id}`;
                  const typeLabel = NOTE_TYPE_LABEL[note.dian_document_type_code] ?? "Nota";
                  const ident = note.prefix || note.number ? ` ${note.prefix ?? ""}${note.number ?? ""}` : " (borrador)";
                  return (
                    <div
                      key={note.id}
                      className="flex items-center justify-between rounded border border-(--border-color) bg-(--bg-secondary) px-3 py-2 text-xs"
                    >
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium text-(--text-primary)">{typeLabel}{ident}</span>
                        <span className="font-mono text-(--text-secondary)">{formatCOP.format(note.payable_cents / 100)}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <StatusBadge status={note.status} />
                        <button
                          type="button"
                          onClick={() => navigate(route)}
                          className="flex items-center gap-1 text-(--accent-primary) hover:underline"
                        >
                          <ExternalLink className="h-3 w-3" />
                          Ver
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          <div className="overflow-x-auto rounded border border-(--border-color)">
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

          <div className="flex flex-wrap justify-end gap-4 text-xs text-(--text-secondary)">
            <span>Subtotal: <span className="font-mono text-(--text-primary)">{formatCOP.format(doc.totals.line_extension_cents / 100)}</span></span>
            <span>Impuestos: <span className="font-mono text-(--text-primary)">{formatCOP.format((doc.totals.tax_inclusive_cents - doc.totals.line_extension_cents) / 100)}</span></span>
            <span className="font-semibold">A pagar: <span className="font-mono text-(--text-primary)">{formatCOP.format(doc.totals.payable_cents / 100)}</span></span>
            {doc.net_payable_cents !== undefined && doc.net_payable_cents !== doc.totals.payable_cents && (
              <span className="font-semibold text-(--accent-primary)">
                Saldo efectivo: <span className="font-mono">{formatCOP.format(doc.net_payable_cents / 100)}</span>
              </span>
            )}
          </div>
        </Card>
      ) : null}
    </div>
  );
}
