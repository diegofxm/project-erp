import { useEffect, useState } from "react";
import { FileMinus, FileText, Mail, Send, Trash2 } from "lucide-react";
import { useNavigate, useParams, useSearchParams } from "react-router";
import {
  confirmDocument,
  createCreditNoteDraft,
  deleteDraft,
  getDocument,
  getDocumentPdfBlobUrl,
  sendDocumentEmail,
  updateCreditNoteDraft,
} from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { dianStatusLabel } from "../lib/dianStatus";
import { useAuth } from "../context/AuthContext";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { BillingReference, Document, IssueCreditNotePayload } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { CreditNoteForm } from "../components/invoice-form/CreditNoteForm";
import { StatusBadge } from "../components/invoice-form/StatusBadge";

// DIAN List 22 — mismo mapa que CreditNoteForm, para mostrar la etiqueta en la vista de solo
// lectura sin duplicar los datos.
const CREDIT_NOTE_TYPE_LABELS: Record<string, string> = {
  "1": "Devolución parcial de los bienes; no aceptación de partes del servicio",
  "2": "Anulación de factura electrónica",
  "3": "Rebaja o descuento parcial",
  "4": "Ajuste de precio",
  "5": "Otros",
};

function billingRefFromDoc(doc: Document): BillingReference {
  return {
    prefix: doc.prefix ?? "",
    number: doc.number?.toString() ?? "",
    cufe: doc.document_key ?? "",
    // issue_date viene como ISO string (ej. "2026-07-09T00:00:00Z"); solo la parte de fecha.
    issue_date: doc.issue_date?.slice(0, 10) ?? "",
  };
}

export function CreditNoteEditorPage() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { activeIssuer } = useAuth();
  const confirmDialog = useConfirm();
  const toast = useToast();
  const isNew = id === "new";
  const sourceInvoiceId = searchParams.get("from");

  const [doc, setDoc] = useState<Document | null>(null);
  const [sourceDoc, setSourceDoc] = useState<Document | null>(null); // factura de origen (solo cuando isNew)
  const [billingRef, setBillingRef] = useState<BillingReference | null>(null);
  const [loadingDocument, setLoadingDocument] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [loadingPdf, setLoadingPdf] = useState(false);
  const [sendingEmail, setSendingEmail] = useState(false);

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
            setError("La factura de referencia debe estar aceptada por la DIAN antes de emitir una nota crédito.");
            return;
          }
          setBillingRef(billingRefFromDoc(src));
          setSourceDoc(src); // pre-llena cliente e ítems en el formulario
        })
        .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la factura de referencia"))
        .finally(() => setLoadingDocument(false));
    } else if (id) {
      getDocument(id)
        .then((d) => {
          setDoc(d);
          if (d.billing_reference) setBillingRef(d.billing_reference);
        })
        .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la nota crédito"))
        .finally(() => setLoadingDocument(false));
    } else {
      setLoadingDocument(false);
    }
  }, [id, isNew, sourceInvoiceId]);

  async function handleSubmit(payload: IssueCreditNotePayload) {
    setError(null);
    setSaving(true);
    try {
      const saved = isNew
        ? await createCreditNoteDraft(payload)
        : await updateCreditNoteDraft(id!, payload);
      if (isNew) {
        navigate(`/documents/credit-notes/${saved.id}`, { replace: true });
      } else {
        setDoc(saved);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar la nota crédito");
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
      navigate("/documents/credit-notes");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el borrador");
    }
  }

  async function handleConfirm() {
    if (!id || isNew) return;
    if (
      !(await confirmDialog(
        "¿Confirmar y enviar esta nota crédito? Reclama el número real ante la DIAN y ya no se puede editar ni eliminar.",
        { confirmLabel: "Confirmar y enviar" }
      ))
    )
      return;
    setConfirming(true);
    try {
      setDoc(await confirmDocument(id));
      toast.success("Nota Crédito confirmada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo confirmar la nota crédito");
    } finally {
      setConfirming(false);
    }
  }

  async function handleViewPdf() {
    if (!id || isNew) return;
    setLoadingPdf(true);
    try {
      const url = await getDocumentPdfBlobUrl(id);
      window.open(url, "_blank");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo generar el PDF");
    } finally {
      setLoadingPdf(false);
    }
  }

  async function handleSendEmail() {
    if (!id || isNew || !doc) return;
    if (!(await confirmDialog(`¿Enviar esta nota crédito por correo a ${doc.customer.email || "el cliente"}?`))) return;
    setSendingEmail(true);
    try {
      await sendDocumentEmail(id);
      toast.success(`Nota Crédito enviada a ${doc.customer.email}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo enviar la nota crédito por correo");
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
    ? "Nueva Nota Crédito"
    : doc?.prefix && doc?.number
    ? `Nota Crédito ${doc.prefix}${doc.number}`
    : "Nota Crédito";

  const showForm = (isNew || doc?.status === "draft") && billingRef;

  return (
    <div className="p-4">
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
            <FileMinus className="h-4 w-4 shrink-0 text-(--accent-primary)" />
            {title}
          </h1>
        <div className="flex items-center gap-2">
          {doc && <StatusBadge status={doc.status} />}
          {!isNew && doc && (
            <Button type="button" variant="secondary" icon={<FileText className="h-3.5 w-3.5" />} loading={loadingPdf} onClick={handleViewPdf}>
              Ver PDF
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
          <CreditNoteForm
            initial={isNew ? null : doc}
            prefill={isNew ? sourceDoc : null}
            billingReference={billingRef}
            onSubmit={handleSubmit}
            onCancel={() => navigate("/documents/credit-notes")}
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
            </div>
            <div className="col-span-3">
              <span className="text-(--text-secondary)">Total</span>
              <p className="font-mono text-(--text-primary)">{formatCOP.format(doc.totals.payable_cents / 100)}</p>
            </div>
            <div className="col-span-3">
              <span className="text-(--text-secondary)">Concepto</span>
              <p className="text-(--text-primary)">
                {doc.note_type_code
                  ? `${doc.note_type_code} — ${CREDIT_NOTE_TYPE_LABELS[doc.note_type_code] ?? ""}`
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

          {(doc.dian_status_code || doc.dian_status_description) && (
            <div className="rounded border border-(--border-color) bg-(--bg-primary) p-3 text-xs">
              <p className="font-medium text-(--text-primary)">Estado DIAN: {dianStatusLabel(doc.dian_status_code)}</p>
              {doc.dian_status_description && (
                <p className="mt-1 text-(--text-secondary)">{doc.dian_status_description}</p>
              )}
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
        </Card>
      ) : null}
    </div>
  );
}
