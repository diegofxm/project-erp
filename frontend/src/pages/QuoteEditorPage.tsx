import { useEffect, useState } from "react";
import { ClipboardList, FileText, Send, Check, X as XIcon, ArrowRightCircle, Trash2 } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import { acceptQuote, convertQuoteToSale, createQuote, deleteQuote, fetchQuote, getQuotePdfBlobUrl, rejectQuote, sendQuoteEmail } from "../lib/quotes";
import { listCustomers } from "../lib/customers";
import { ApiError } from "../lib/apiClient";
import { openInNewTab } from "../lib/openInNewTab";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import { formatCOP } from "../lib/currency";
import { formatDateOnly } from "../lib/dateFormat";
import type { Customer, Quote, QuoteStatus, SalesLineInput } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Combobox } from "../components/ui/Combobox";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { SendEmailModal } from "../components/ui/SendEmailModal";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";
import { SalesLineItemsEditor, salesLinesTotal } from "../components/sales/SalesLineItemsEditor";

const STATUS_LABEL: Record<QuoteStatus, string> = {
  draft: "Borrador", sent: "Enviada", accepted: "Aceptada", rejected: "Rechazada", expired: "Vencida",
};
const STATUS_TONE: Record<QuoteStatus, StatusTone> = {
  draft: "neutral", sent: "info", accepted: "success", rejected: "danger", expired: "warning",
};

// A diferencia de los editores de documentos electrónicos, sales/ NO tiene endpoint de
// actualización (ver application/quote.go — solo Create/Send/Accept/Reject/ConvertToSale/
// Delete). Una vez creada, la cotización es de solo lectura salvo por sus transiciones de
// estado: no hay "editar borrador ya guardado", solo "crear" o "avanzar/rechazar/eliminar".
export function QuoteEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const confirmDialog = useConfirm();
  const toast = useToast();
  const isNew = id === "new";

  const [quote, setQuote] = useState<Quote | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [loadingPdf, setLoadingPdf] = useState(false);
  const [showEmailModal, setShowEmailModal] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [customerId, setCustomerId] = useState("");
  const [validUntil, setValidUntil] = useState("");
  const [notes, setNotes] = useState("");
  const [lines, setLines] = useState<SalesLineInput[]>([]);

  useEffect(() => {
    listCustomers().then(setCustomers).catch(() => setCustomers([]));
  }, []);

  useEffect(() => {
    if (isNew || !id) return;
    setLoading(true);
    fetchQuote(id)
      .then(setQuote)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la cotización"))
      .finally(() => setLoading(false));
  }, [id, isNew]);

  const customerOptions = customers.map((c) => ({ value: c.id, label: c.name }));
  const customerName = (cid: string) => customers.find((c) => c.id === cid)?.name ?? "—";

  async function handleCreate() {
    setError(null);
    setSaving(true);
    try {
      const created = await createQuote({
        customer_id: customerId,
        lines,
        valid_until: validUntil || undefined,
        notes,
      });
      toast.success(`Cotización ${created.number || ""} creada.`);
      navigate(`/sales/quotes/${created.id}`, { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo crear la cotización");
    } finally {
      setSaving(false);
    }
  }

  async function handleViewPdf() {
    if (!quote) return;
    setLoadingPdf(true);
    try {
      const url = await getQuotePdfBlobUrl(quote.id);
      openInNewTab(url);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo generar el PDF");
    } finally {
      setLoadingPdf(false);
    }
  }

  // El backend de cotizaciones aún no soporta CC (ver sales/application/quote_email.go) — se
  // ignora acá también, igual que el resto del modal reutilizado de InvoiceEditorPage.
  async function handleSendEmailConfirm(to: string, _cc: string[]) {
    if (!quote) return;
    try {
      setQuote(await sendQuoteEmail(quote.id, to));
      toast.success(`Cotización enviada a ${to}.`);
      setShowEmailModal(false);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo enviar la cotización");
      throw err;
    }
  }

  async function handleAccept() {
    if (!quote) return;
    setSaving(true);
    try {
      setQuote(await acceptQuote(quote.id));
      toast.success("Cotización aceptada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo aceptar la cotización");
    } finally {
      setSaving(false);
    }
  }

  async function handleReject() {
    if (!quote) return;
    if (!(await confirmDialog("¿Rechazar esta cotización? No se puede deshacer.", { tone: "danger" }))) return;
    setSaving(true);
    try {
      await rejectQuote(quote.id);
      setQuote({ ...quote, status: "rejected" });
      toast.success("Cotización rechazada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo rechazar la cotización");
    } finally {
      setSaving(false);
    }
  }

  async function handleConvert() {
    if (!quote) return;
    setSaving(true);
    try {
      const sale = await convertQuoteToSale(quote.id);
      toast.success(`Convertida en venta ${sale.number || ""}.`);
      navigate(`/sales/${sale.id}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo convertir a venta");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!quote) return;
    if (!(await confirmDialog(`¿Eliminar la cotización ${quote.number || "en borrador"}? Esta acción no se puede deshacer.`, { tone: "danger" }))) return;
    try {
      await deleteQuote(quote.id);
      toast.success("Cotización eliminada.");
      navigate("/sales/quotes");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar la cotización");
    }
  }

  if (loading) {
    return <div className="flex min-h-32 items-center justify-center p-4"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>;
  }

  const title = isNew ? "Nueva cotización" : quote ? `Cotización ${quote.number || "(borrador)"}` : "Cotización";
  const total = isNew ? salesLinesTotal(lines) : quote ? salesLinesTotal(quote.lines) : 0;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Ventas", to: "/sales" }, { label: "Cotizaciones", to: "/sales/quotes" }, quote ? { label: quote.number || "(borrador)", muted: true } : { label: title }]} />
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <ClipboardList className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          {title}
        </h1>
        <div className="flex flex-wrap items-center gap-1.5">
          {quote && <StatusPill tone={STATUS_TONE[quote.status]} label={STATUS_LABEL[quote.status]} />}
          {isNew && (
            <Button type="button" loading={saving} disabled={!customerId || lines.length === 0} onClick={handleCreate}>
              Guardar cotización
            </Button>
          )}
          {!isNew && quote && (
            <Button type="button" variant="secondary" icon={<FileText className="h-3.5 w-3.5" />} loading={loadingPdf} onClick={handleViewPdf}>
              Ver PDF
            </Button>
          )}
          {quote?.status === "draft" && (
            <>
              <Button type="button" variant="danger" icon={<Trash2 className="h-3.5 w-3.5" />} onClick={handleDelete}>Eliminar</Button>
              <Button type="button" icon={<Send className="h-3.5 w-3.5" />} onClick={() => setShowEmailModal(true)}>Enviar</Button>
            </>
          )}
          {quote?.status === "sent" && (
            <>
              <Button type="button" variant="danger" icon={<XIcon className="h-3.5 w-3.5" />} loading={saving} onClick={handleReject}>Rechazar</Button>
              <Button type="button" variant="success" icon={<Check className="h-3.5 w-3.5" />} loading={saving} onClick={handleAccept}>Aceptar</Button>
            </>
          )}
          {quote?.status === "accepted" && (
            <Button type="button" icon={<ArrowRightCircle className="h-3.5 w-3.5" />} loading={saving} onClick={handleConvert}>
              Convertir a venta
            </Button>
          )}
        </div>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      <Card className="p-4">
        <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
          {isNew ? (
            <>
              <Combobox label="Cliente" value={customerId} onChange={setCustomerId} options={customerOptions} placeholder="Buscar cliente…" />
              <Input label="Válida hasta (opcional)" type="date" value={validUntil} onChange={(e) => setValidUntil(e.target.value)} />
              <Input label="Notas" value={notes} onChange={(e) => setNotes(e.target.value)} />
            </>
          ) : quote ? (
            <>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Cliente</p>
                <p className="text-xs text-(--text-primary)">{customerName(quote.customer_id)}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Fecha</p>
                <p className="text-xs text-(--text-primary)">{formatDateOnly(quote.issue_date)}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Válida hasta</p>
                <p className="text-xs text-(--text-primary)">{quote.valid_until ? formatDateOnly(quote.valid_until) : "—"}</p>
              </div>
              {quote.notes && (
                <div className="sm:col-span-3">
                  <p className="text-xs font-medium text-(--text-secondary)">Notas</p>
                  <p className="text-xs text-(--text-primary)">{quote.notes}</p>
                </div>
              )}
            </>
          ) : null}
        </div>

        <SalesLineItemsEditor lines={isNew ? lines : quote?.lines ?? []} onChange={setLines} disabled={!isNew} />

        <div className="mt-4 flex justify-end border-t border-(--border-light) pt-3">
          <span className="text-xs text-(--text-secondary)">
            Total: <span className="font-mono text-sm font-semibold text-(--text-primary)">{formatCOP.format(total)}</span>
          </span>
        </div>
      </Card>

      {showEmailModal && quote && (
        <SendEmailModal
          initialEmail={customers.find((c) => c.id === quote.customer_id)?.email ?? ""}
          onSend={handleSendEmailConfirm}
          onClose={() => setShowEmailModal(false)}
        />
      )}
    </div>
  );
}
