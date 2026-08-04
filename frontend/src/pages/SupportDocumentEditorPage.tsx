import { useEffect, useState } from "react";
import { BookCopy, ExternalLink, FileCode, FileDiff, FileText, Mail, Send, Trash2 } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import {
  confirmDocument,
  createSupportDocumentDraft,
  deleteDraft,
  getDocument,
  getDocumentPdfBlobUrl,
  getDocumentXmlBlobUrl,
  sendDocumentEmail,
  updateSupportDocumentDraft,
} from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { openInNewTab } from "../lib/openInNewTab";
import { formatCOP } from "../lib/currency";
import { formatDateOnly } from "../lib/dateFormat";
import { idTypeLabel } from "../lib/idTypes";
import { useAuth } from "../context/AuthContext";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import { SendEmailModal } from "../components/ui/SendEmailModal";
import { fetchSupplier } from "../lib/suppliers";
import { usePdfFormat } from "../lib/usePdfFormat";
import type { Document, IssueSupportDocumentPayload } from "../lib/types";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { Banner } from "../components/ui/Banner";
import { DianStatusBlock } from "../components/DianStatusBlock";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { SupportDocumentForm } from "../components/invoice-form/SupportDocumentForm";
import { StatusBadge } from "../components/invoice-form/StatusBadge";

const OPERATION_LABELS: Record<string, string> = { "10": "Residente", "11": "No Residente" };

export function SupportDocumentEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { activeCompany } = useAuth();
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
  const [showEmailModal, setShowEmailModal] = useState(false);
  const [pdfFormat] = usePdfFormat();

  useEffect(() => {
    if (isNew || !id) return;
    getDocument(id)
      .then(setDoc)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el documento soporte"))
      .finally(() => setLoadingDocument(false));
  }, [id, isNew]);

  async function handleSubmit(payload: IssueSupportDocumentPayload) {
    setError(null);
    setSaving(true);
    try {
      const saved = isNew
        ? await createSupportDocumentDraft(payload)
        : await updateSupportDocumentDraft(id!, payload);
      if (isNew) {
        navigate(`/documents/support-documents/${saved.id}`, { replace: true });
      } else {
        setDoc(saved);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar el documento soporte");
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
      navigate("/documents/support-documents");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el borrador");
    }
  }

  async function handleConfirm() {
    if (!id || isNew) return;
    if (
      !(await confirmDialog(
        "¿Confirmar y enviar este Documento Soporte? Reclama el número real ante la DIAN y ya no se puede editar ni eliminar.",
        { confirmLabel: "Confirmar y enviar" }
      ))
    )
      return;
    setConfirming(true);
    try {
      setDoc(await confirmDocument(id));
      toast.success("Documento Soporte confirmado.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo confirmar el documento soporte");
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

  async function handleSendEmailConfirm(to: string, cc: string[]) {
    if (!id || isNew || !doc) return;
    try {
      await sendDocumentEmail(id, pdfFormat, to, cc);
      toast.success(`Documento Soporte enviado a ${to}`);
      setShowEmailModal(false);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo enviar el documento por correo");
      throw err;
    }
  }

  async function handleDownloadXml() {
    if (!id || isNew) return;
    setLoadingXml(true);
    try {
      const url = await getDocumentXmlBlobUrl(id);
      const a = document.createElement("a");
      a.href = url;
      a.download = doc ? `${doc.prefix ?? "DS"}${doc.number ?? ""}.xml` : "documento-soporte.xml";
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo descargar el XML");
    } finally {
      setLoadingXml(false);
    }
  }

  const companyNotReady = !activeCompany?.has_software_credentials || !activeCompany?.has_certificate;

  if (loadingDocument) {
    return (
      <div className="flex min-h-32 items-center justify-center p-4">
        <Spinner className="h-5 w-5 text-(--text-muted)" />
      </div>
    );
  }

  const title = isNew
    ? "Nuevo Documento Soporte"
    : doc?.prefix && doc?.number
    ? `Doc. Soporte ${doc.prefix}${doc.number}`
    : "Documento Soporte";

  const showForm = isNew || doc?.status === "draft";

  const withholdingTotal = doc?.withholding_taxes?.reduce((s, t) => s + t.tax_amount_cents, 0) ?? 0;
  const netPayable = (doc?.totals.payable_cents ?? 0) - withholdingTotal;

  return (
    <div className="p-4">
      <Breadcrumbs
        items={[{ label: "Documentos" }, { label: "Documento Soporte", to: "/documents/support-documents" }, { label: title }]}
      />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <BookCopy className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          {title}
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
            <Button type="button" variant="secondary" icon={<Mail className="h-3.5 w-3.5" />} onClick={() => setShowEmailModal(true)}>
              Enviar al proveedor
            </Button>
          )}
          {!isNew && doc?.status === "accepted" && (() => {
            const draftNA = doc.related_notes?.find(n => n.dian_document_type_code === "95" && n.status === "draft");
            return (
              <Button
                type="button"
                variant="secondary"
                icon={<FileDiff className="h-3.5 w-3.5" />}
                onClick={() => navigate(draftNA ? `/documents/adjustment-notes/${draftNA.id}` : `/documents/adjustment-notes/new?from=${id}`)}
              >
                {draftNA ? "Editar borrador NA" : "Emitir Nota de Ajuste"}
              </Button>
            );
          })()}
          {!isNew && doc?.status === "draft" && (
            <>
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
                disabled={companyNotReady}
                loading={confirming}
              >
                Confirmar y enviar
              </Button>
            </>
          )}
        </div>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {!isNew && doc?.status === "draft" && companyNotReady && (
        <Banner tone="info">
          La empresa todavía no tiene software/certificado configurados — complétalos en Configuración → Empresa antes de confirmar.
        </Banner>
      )}

      {showForm ? (
        <Card className="mt-3">
          <SupportDocumentForm
            initial={isNew ? null : doc}
            onSubmit={handleSubmit}
            onCancel={() => navigate("/documents/support-documents")}
            loading={saving}
          />
        </Card>
      ) : doc ? (
        <Card className="mt-3 flex flex-col gap-4 p-4">
          <div className="grid grid-cols-12 gap-3 text-xs">
            <div className="col-span-6 sm:col-span-3">
              <span className="text-(--text-secondary)">Tercero (vendedor)</span>
              <p className="text-(--text-primary)">{doc.supplier?.name ?? "—"}</p>
              <p className="text-(--text-muted)">
                {doc.supplier ? `${idTypeLabel(doc.supplier.identification.type_code)} ${doc.supplier.identification.number}` : ""}
              </p>
            </div>
            <div className="col-span-6 sm:col-span-3">
              <span className="text-(--text-secondary)">Tipo de operación</span>
              <p className="text-(--text-primary)">
                {doc.operation_type_code ? `${OPERATION_LABELS[doc.operation_type_code] ?? ""} (${doc.operation_type_code})` : "—"}
              </p>
            </div>
            <div className="col-span-6 sm:col-span-3">
              <span className="text-(--text-secondary)">A pagar (neto)</span>
              <p className="font-mono text-(--text-primary)">{formatCOP.format(netPayable / 100)}</p>
            </div>
            <div className="col-span-6 sm:col-span-3">
              <span className="text-(--text-secondary)">Fecha de emisión</span>
              <p className="text-(--text-primary)">
                {doc.issue_date ? formatDateOnly(doc.issue_date) : "—"}
              </p>
            </div>
            <div className="col-span-12">
              <span className="text-(--text-secondary)">CUDS</span>
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
                Notas de Ajuste emitidas ({doc.related_notes.length})
              </p>
              <div className="flex flex-col gap-1">
                {doc.related_notes.map((note) => {
                  const ident = note.prefix || note.number ? ` ${note.prefix ?? ""}${note.number ?? ""}` : " (borrador)";
                  return (
                    <div
                      key={note.id}
                      className="flex items-center justify-between rounded border border-(--border-color) bg-(--bg-secondary) px-3 py-2 text-xs"
                    >
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium text-(--text-primary)">Nota de Ajuste{ident}</span>
                        <span className="font-mono text-(--text-secondary)">{formatCOP.format(note.payable_cents / 100)}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <StatusBadge status={note.status} />
                        <button
                          type="button"
                          onClick={() => navigate(`/documents/adjustment-notes/${note.id}`)}
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

          {doc.withholding_taxes && doc.withholding_taxes.length > 0 && (
            <div>
              <h3 className="mb-2 text-xs font-semibold text-(--text-primary)">Retenciones</h3>
              <div className="overflow-x-auto rounded border border-(--border-color)">
                <table className="w-full text-left text-xs">
                  <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                    <tr>
                      <th className="px-3 py-2 font-medium">Tipo</th>
                      <th className="px-3 py-2 font-medium">Base</th>
                      <th className="px-3 py-2 font-medium">Retención</th>
                      <th className="px-3 py-2 font-medium">%</th>
                    </tr>
                  </thead>
                  <tbody>
                    {doc.withholding_taxes.map((t, i) => (
                      <tr key={i} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                        <td className="px-3 py-2 text-(--text-primary)">{t.type_name} ({t.type_code})</td>
                        <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(t.taxable_amount_cents / 100)}</td>
                        <td className="px-3 py-2 font-mono text-(--danger)">{formatCOP.format(t.tax_amount_cents / 100)}</td>
                        <td className="px-3 py-2 font-mono text-(--text-secondary)">{t.percent}%</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
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
                      {formatCOP.format(
                        (line.line_extension_cents + (line.taxes?.reduce((s, t) => s + t.tax_amount_cents, 0) ?? 0)) / 100
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="flex flex-wrap justify-end gap-4 text-xs text-(--text-secondary)">
            <span>Subtotal: <span className="font-mono text-(--text-primary)">{formatCOP.format(doc.totals.line_extension_cents / 100)}</span></span>
            <span>Impuestos: <span className="font-mono text-(--text-primary)">{formatCOP.format((doc.totals.tax_inclusive_cents - doc.totals.line_extension_cents) / 100)}</span></span>
            {withholdingTotal > 0 && (
              <span>Retenciones: <span className="font-mono text-(--danger)">− {formatCOP.format(withholdingTotal / 100)}</span></span>
            )}
            <span className="font-semibold">A pagar: <span className="font-mono text-(--text-primary)">{formatCOP.format(netPayable / 100)}</span></span>
            {doc.net_payable_cents !== undefined && doc.net_payable_cents !== doc.totals.payable_cents && (
              <span className="font-semibold text-(--accent-primary)">
                Saldo efectivo: <span className="font-mono">{formatCOP.format(doc.net_payable_cents / 100)}</span>
              </span>
            )}
          </div>
        </Card>
      ) : null}
      {showEmailModal && doc && (
        <SendEmailModal
          initialEmail={doc.supplier?.email ?? ""}
          fetchEmail={doc.supplier_id ? () => fetchSupplier(doc.supplier_id!).then((v) => v.email ?? "") : undefined}
          onSend={handleSendEmailConfirm}
          onClose={() => setShowEmailModal(false)}
        />
      )}
    </div>
  );
}
