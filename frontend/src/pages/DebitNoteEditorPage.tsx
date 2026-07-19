import { useEffect, useState } from "react";
import { FileCode, FilePlus, FileText, Mail, Send, Trash2 } from "lucide-react";
import { useNavigate, useParams, useSearchParams } from "react-router";
import {
  confirmDocument,
  createDebitNoteDraft,
  deleteDraft,
  getDocument,
  getDocumentPdfBlobUrl,
  getDocumentXmlBlobUrl,
  sendDocumentEmail,
  updateDebitNoteDraft,
} from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { idTypeLabel } from "../lib/idTypes";
import { useAuth } from "../context/AuthContext";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import { usePdfFormat } from "../lib/usePdfFormat";
import type { BillingReference, Document, IssueDebitNotePayload } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { DianStatusBlock } from "../components/DianStatusBlock";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { DebitNoteForm } from "../components/invoice-form/DebitNoteForm";
import { StatusBadge } from "../components/invoice-form/StatusBadge";

// DIAN — Códigos de concepto para Nota Débito (mismo mapa que DebitNoteForm, para la vista
// de solo lectura sin duplicar los datos).
const DEBIT_NOTE_TYPE_LABELS: Record<string, string> = {
  "1": "Intereses",
  "2": "Gastos por cobrar",
  "3": "Cambio del valor",
};

function billingRefFromDoc(doc: Document): BillingReference {
  return {
    prefix: doc.prefix ?? "",
    number: doc.number?.toString() ?? "",
    cufe: doc.document_key ?? "",
    issue_date: doc.issue_date?.slice(0, 10) ?? "",
  };
}

export function DebitNoteEditorPage() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { activeIssuer } = useAuth();
  const confirmDialog = useConfirm();
  const toast = useToast();
  const isNew = id === "new";
  const sourceInvoiceId = searchParams.get("from");

  const [doc, setDoc] = useState<Document | null>(null);
  const [sourceDoc, setSourceDoc] = useState<Document | null>(null);
  const [billingRef, setBillingRef] = useState<BillingReference | null>(null);
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
      if (!sourceInvoiceId) {
        setError("Falta el parámetro ?from= con el ID de la factura de referencia.");
        setLoadingDocument(false);
        return;
      }
      getDocument(sourceInvoiceId)
        .then((src) => {
          if (src.status !== "accepted") {
            setError("La factura de referencia debe estar aceptada por la DIAN antes de emitir una nota débito.");
            return;
          }
          setBillingRef(billingRefFromDoc(src));
          setSourceDoc(src);
        })
        .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la factura de referencia"))
        .finally(() => setLoadingDocument(false));
    } else if (id) {
      getDocument(id)
        .then((d) => {
          setDoc(d);
          if (d.billing_reference) setBillingRef(d.billing_reference);
        })
        .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la nota débito"))
        .finally(() => setLoadingDocument(false));
    } else {
      setLoadingDocument(false);
    }
  }, [id, isNew, sourceInvoiceId]);

  async function handleSubmit(payload: IssueDebitNotePayload) {
    setError(null);
    setSaving(true);
    try {
      const saved = isNew
        ? await createDebitNoteDraft(payload)
        : await updateDebitNoteDraft(id!, payload);
      if (isNew) {
        navigate(`/documents/debit-notes/${saved.id}`, { replace: true });
      } else {
        setDoc(saved);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar la nota débito");
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
      navigate("/documents/debit-notes");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el borrador");
    }
  }

  async function handleConfirm() {
    if (!id || isNew) return;
    if (
      !(await confirmDialog(
        "¿Confirmar y enviar esta nota débito? Reclama el número real ante la DIAN y ya no se puede editar ni eliminar.",
        { confirmLabel: "Confirmar y enviar" }
      ))
    )
      return;
    setConfirming(true);
    try {
      setDoc(await confirmDocument(id));
      toast.success("Nota Débito confirmada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo confirmar la nota débito");
    } finally {
      setConfirming(false);
    }
  }

  async function handleViewPdf() {
    if (!id || isNew) return;
    setLoadingPdf(true);
    try {
      const url = await getDocumentPdfBlobUrl(id, pdfFormat);
      window.open(url, "_blank");
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
      a.download = doc ? `${doc.prefix ?? ""}${doc.number ?? ""}.xml` : "nota-debito.xml";
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo descargar el XML");
    } finally {
      setLoadingXml(false);
    }
  }

  async function handleSendEmail() {
    if (!id || isNew || !doc) return;
    if (!(await confirmDialog(`¿Enviar esta nota débito por correo a ${doc.customer.email || "el cliente"}?`))) return;
    setSendingEmail(true);
    try {
      await sendDocumentEmail(id, pdfFormat);
      toast.success(`Nota Débito enviada a ${doc.customer.email}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo enviar la nota débito por correo");
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

  const title = isNew
    ? "Nueva Nota Débito"
    : doc?.prefix && doc?.number
    ? `Nota Débito ${doc.prefix}${doc.number}`
    : "Nota Débito";

  const showForm = (isNew || doc?.status === "draft") && billingRef;

  return (
    <div className="p-4">
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
            <FilePlus className="h-4 w-4 shrink-0 text-(--accent-primary)" />
            {title}
          </h1>
        <div className="flex items-center gap-2">
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
          La empresa todavía no tiene software/certificado configurados — complétalos en Configuración → Empresa antes
          de confirmar.
        </Banner>
      )}

      {showForm ? (
        <Card className="mt-3">
          <DebitNoteForm
            initial={isNew ? null : doc}
            prefill={isNew ? sourceDoc : null}
            billingReference={billingRef}
            onSubmit={handleSubmit}
            onCancel={() => navigate("/documents/debit-notes")}
            loading={saving}
          />
        </Card>
      ) : doc ? (
        <Card className="mt-3 flex flex-col gap-4 p-4">
          {billingRef && (
            <div>
              <span className="text-xs text-(--text-secondary)">Factura referenciada</span>
              <p className="font-mono text-sm text-(--text-primary)">
                {billingRef.prefix}{billingRef.number}
              </p>
            </div>
          )}

          <div className="grid grid-cols-12 gap-3 text-xs">
            <div className="col-span-3">
              <span className="text-(--text-secondary)">Cliente</span>
              <p className="text-(--text-primary)">{doc.customer.name}</p>
              <p className="text-(--text-muted)">{idTypeLabel(doc.customer.identification.type_code)} {doc.customer.identification.number}</p>
            </div>
            <div className="col-span-3">
              <span className="text-(--text-secondary)">Total</span>
              <p className="font-mono text-(--text-primary)">{formatCOP.format(doc.totals.payable_cents / 100)}</p>
            </div>
            <div className="col-span-3">
              <span className="text-(--text-secondary)">Concepto</span>
              <p className="text-(--text-primary)">
                {doc.discrepancy_response?.response_code
                  ? `${doc.discrepancy_response.response_code} — ${DEBIT_NOTE_TYPE_LABELS[doc.discrepancy_response.response_code] ?? ""}`
                  : "—"}
              </p>
            </div>
            <div className="col-span-3">
              <span className="text-(--text-secondary)">Fecha de emisión</span>
              <p className="text-(--text-primary)">
                {doc.issue_date ? new Date(doc.issue_date).toLocaleDateString("es-CO") : "—"}
              </p>
            </div>
            <div className="col-span-12">
              <span className="text-(--text-secondary)">CUDE</span>
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
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">
                      {formatCOP.format(line.unit_price_cents / 100)}
                    </td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">
                      {formatCOP.format(
                        (line.line_extension_cents + (line.taxes?.reduce((s, t) => s + t.tax_amount_cents, 0) ?? 0)) /
                          100
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="flex justify-end gap-4 text-xs text-(--text-secondary)">
            <span>Subtotal: <span className="font-mono text-(--text-primary)">{formatCOP.format(doc.totals.line_extension_cents / 100)}</span></span>
            <span>Impuestos: <span className="font-mono text-(--text-primary)">{formatCOP.format((doc.totals.tax_inclusive_cents - doc.totals.line_extension_cents) / 100)}</span></span>
            <span className="font-semibold">A pagar: <span className="font-mono text-(--text-primary)">{formatCOP.format(doc.totals.payable_cents / 100)}</span></span>
          </div>
        </Card>
      ) : null}
    </div>
  );
}
