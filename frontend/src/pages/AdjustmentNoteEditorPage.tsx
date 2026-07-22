import { useEffect, useState } from "react";
import { FileDiff, FileCode, FileText, Mail, Send, Trash2 } from "lucide-react";
import { useNavigate, useParams, useSearchParams } from "react-router";
import {
  confirmDocument,
  createAdjustmentNoteDraft,
  deleteDraft,
  getDocument,
  getDocumentPdfBlobUrl,
  getDocumentXmlBlobUrl,
  sendDocumentEmail,
  updateAdjustmentNoteDraft,
} from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { openInNewTab } from "../lib/openInNewTab";
import { formatCOP } from "../lib/currency";
import { idTypeLabel } from "../lib/idTypes";
import { useAuth } from "../context/AuthContext";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import { usePdfFormat } from "../lib/usePdfFormat";
import type { BillingReference, Document, IssueAdjustmentNotePayload } from "../lib/types";
import { BackLink } from "../components/ui/BackLink";
import { Banner } from "../components/ui/Banner";
import { SourceBalanceBlock } from "../components/ui/SourceBalanceBlock";
import { DianStatusBlock } from "../components/DianStatusBlock";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { AdjustmentNoteForm } from "../components/invoice-form/AdjustmentNoteForm";
import { StatusBadge } from "../components/invoice-form/StatusBadge";

const OPERATION_LABELS: Record<string, string> = { "10": "Residente", "11": "No Residente" };

function billingRefFromDoc(doc: Document): BillingReference {
  return {
    prefix: doc.prefix ?? "",
    number: doc.number?.toString() ?? "",
    cufe: doc.document_key ?? "",
    issue_date: doc.issue_date?.slice(0, 10) ?? "",
  };
}

export function AdjustmentNoteEditorPage() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { activeIssuer } = useAuth();
  const confirmDialog = useConfirm();
  const toast = useToast();
  const isNew = id === "new";
  const sourceDocId = searchParams.get("from");

  const [doc, setDoc] = useState<Document | null>(null);
  const [sourceDoc, setSourceDoc] = useState<Document | null>(null);
  const [billingRef, setBillingRef] = useState<BillingReference | null>(null);
  const [pendingCents, setPendingCents] = useState(0);
  const [loadingSourceDoc, setLoadingSourceDoc] = useState(false);
  const [loadingDocument, setLoadingDocument] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [loadingPdf, setLoadingPdf] = useState(false);
  const [loadingXml, setLoadingXml] = useState(false);
  const [sendingEmail, setSendingEmail] = useState(false);
  const [pdfFormat] = usePdfFormat();

  useEffect(() => {
    if (isNew) {
      if (!sourceDocId) {
        setError("Falta el parámetro ?from= con el ID del Documento Soporte de referencia.");
        setLoadingDocument(false);
        return;
      }
      getDocument(sourceDocId)
        .then((src) => {
          if (src.status !== "accepted") {
            setError("El Documento Soporte debe estar aceptado por la DIAN antes de emitir una nota de ajuste.");
            return;
          }
          setBillingRef(billingRefFromDoc(src));
          setSourceDoc(src);
        })
        .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el Documento Soporte de referencia"))
        .finally(() => setLoadingDocument(false));
    } else if (id) {
      getDocument(id)
        .then((d) => {
          setDoc(d);
          if (d.billing_reference) setBillingRef(d.billing_reference);
          if (d.source_document_id) {
            setLoadingSourceDoc(true);
            getDocument(d.source_document_id)
              .then(setSourceDoc)
              .catch(() => {})
              .finally(() => setLoadingSourceDoc(false));
          }
        })
        .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la nota de ajuste"))
        .finally(() => setLoadingDocument(false));
    } else {
      setLoadingDocument(false);
    }
  }, [id, isNew, sourceDocId]);

  async function handleSubmit(payload: IssueAdjustmentNotePayload) {
    setError(null);
    setSaving(true);
    try {
      const saved = isNew
        ? await createAdjustmentNoteDraft(payload)
        : await updateAdjustmentNoteDraft(id!, payload);
      if (isNew) {
        navigate(`/documents/adjustment-notes/${saved.id}`, { replace: true });
      } else {
        setDoc(saved);
        if (sourceDoc?.id) getDocument(sourceDoc.id).then(setSourceDoc).catch(() => {});
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar la nota de ajuste");
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
      navigate("/documents/adjustment-notes");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el borrador");
    }
  }

  async function handleConfirm() {
    if (!id || isNew) return;
    if (
      !(await confirmDialog(
        "¿Confirmar y enviar esta Nota de Ajuste? Reclama el número real ante la DIAN y ya no se puede editar ni eliminar.",
        { confirmLabel: "Confirmar y enviar" }
      ))
    )
      return;
    setConfirming(true);
    try {
      setDoc(await confirmDocument(id));
      toast.success("Nota de Ajuste confirmada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo confirmar la nota de ajuste");
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

  async function handleSendEmail() {
    if (!id || isNew || !doc) return;
    if (!(await confirmDialog(`¿Enviar esta Nota de Ajuste por correo a ${doc.vendor?.email || "el proveedor"}?`))) return;
    setSendingEmail(true);
    try {
      await sendDocumentEmail(id, pdfFormat);
      toast.success(`Nota de Ajuste enviada a ${doc.vendor?.email}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo enviar el documento por correo");
    } finally {
      setSendingEmail(false);
    }
  }

  async function handleDownloadXml() {
    if (!id || isNew) return;
    setLoadingXml(true);
    try {
      const url = await getDocumentXmlBlobUrl(id);
      const a = document.createElement("a");
      a.href = url;
      a.download = doc ? `${doc.prefix ?? "NA"}${doc.number ?? ""}.xml` : "nota-ajuste.xml";
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo descargar el XML");
    } finally {
      setLoadingXml(false);
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

  const title = isNew
    ? "Nueva Nota de Ajuste"
    : doc?.prefix && doc?.number
    ? `Nota de Ajuste ${doc.prefix}${doc.number}`
    : "Nota de Ajuste";

  const showForm = (isNew || doc?.status === "draft") && billingRef;

  const withholdingTotal = doc?.withholding_taxes?.reduce((s, t) => s + t.tax_amount_cents, 0) ?? 0;
  const netPayable = (doc?.totals.payable_cents ?? 0) - withholdingTotal;

  return (
    <div className="p-4">
      <BackLink
        to={isNew && sourceDocId ? `/documents/support-documents/${sourceDocId}` : "/documents/adjustment-notes"}
        label={isNew && sourceDoc ? `DS ${sourceDoc.prefix ?? ""}${sourceDoc.number ?? ""}` : "Notas de Ajuste"}
      />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <FileDiff className="h-4 w-4 shrink-0 text-(--accent-primary)" />
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
            <Button type="button" variant="secondary" icon={<Mail className="h-3.5 w-3.5" />} loading={sendingEmail} onClick={handleSendEmail}>
              Enviar al proveedor
            </Button>
          )}
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

      {loadingSourceDoc && (
        <div className="mt-3 flex items-center justify-center rounded border border-(--color-info-border) bg-(--color-info-bg) p-4">
          <Spinner className="h-4 w-4 text-(--color-info-border)" />
        </div>
      )}
      {!loadingSourceDoc && sourceDoc && (
        <SourceBalanceBlock
          doc={sourceDoc}
          pendingCents={pendingCents > 0 ? pendingCents : undefined}
          pendingTypeCode="95"
          editingNoteId={!isNew ? id : undefined}
        />
      )}

      {showForm ? (
        <Card className="mt-3">
          <AdjustmentNoteForm
            initial={isNew ? null : doc}
            prefill={isNew ? sourceDoc : null}
            billingReference={billingRef!}
            onSubmit={handleSubmit}
            onCancel={() => navigate(isNew && sourceDocId ? `/documents/support-documents/${sourceDocId}` : "/documents/adjustment-notes")}
            loading={saving}
            onTotalChange={setPendingCents}
          />
        </Card>
      ) : doc ? (
        <Card className="mt-3 flex flex-col gap-4 p-4">
          {/* Motivo del ajuste */}
          {doc.discrepancy_response && (
            <p className="text-xs text-(--text-muted)">
              Motivo {doc.discrepancy_response.response_code}: {doc.discrepancy_response.description}
            </p>
          )}

          <div className="grid grid-cols-12 gap-3 text-xs">
            <div className="col-span-6 sm:col-span-3">
              <span className="text-(--text-secondary)">Tercero (vendedor)</span>
              <p className="text-(--text-primary)">{doc.vendor?.name ?? "—"}</p>
              <p className="text-(--text-muted)">
                {doc.vendor ? `${idTypeLabel(doc.vendor.identification.type_code)} ${doc.vendor.identification.number}` : ""}
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
                {doc.issue_date ? new Date(doc.issue_date).toLocaleDateString("es-CO") : "—"}
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
          </div>
        </Card>
      ) : null}
    </div>
  );
}
