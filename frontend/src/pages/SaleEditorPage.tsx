import { useEffect, useState } from "react";
import { ShoppingCart, Check, Ban, FileText, Trash2 } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import { cancelSale, confirmSale, createSale, deleteSale, fetchSale, updateSale } from "../lib/sales";
import { listCustomers } from "../lib/customers";
import { listNumberingRanges } from "../lib/numberingRanges";
import { createInvoiceFromSale, getDocument } from "../lib/documents";
import { listPaymentMethods, listPaymentTerms } from "../lib/catalogs";
import { useCatalog } from "../lib/useCatalog";
import { ApiError } from "../lib/apiClient";
import { useConfirm } from "../context/ConfirmContext";
import { useCanManage } from "../hooks/useCanManage";
import { useToast } from "../context/ToastContext";
import { formatDateOnly, todayColombiaISO } from "../lib/dateFormat";
import type { Customer, DocumentStatus, NumberingRange, PaymentMean, Sale, SaleStatus, SalesLineInput } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Combobox } from "../components/ui/Combobox";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";
import { SalesLineItemsEditor } from "../components/sales/SalesLineItemsEditor";
import { SalesTotalsSummary } from "../components/sales/SalesTotalsSummary";
import { PaymentMeansEditor } from "../components/invoice-form/PaymentMeansEditor";

const STATUS_LABEL: Record<SaleStatus, string> = { draft: "Borrador", confirmed: "Confirmada", cancelled: "Cancelada" };
const STATUS_TONE: Record<SaleStatus, StatusTone> = { draft: "neutral", confirmed: "success", cancelled: "danger" };
const INVOICE_DIAN_DOCUMENT_TYPE = "01";
const today = () => todayColombiaISO();
// Mismo criterio que erp/internal/electronic/application/confirm.go isRetryableFailure -- estados
// terminales donde el consecutivo ya se liberó, así que es seguro generar una factura nueva desde
// la venta en vez de quedar bloqueada para siempre apuntando a la factura rechazada.
const RETRYABLE_INVOICE_STATUSES: DocumentStatus[] = ["rejected", "send_error", "environment_mismatch"];

// Editable mientras está en borrador (crear o corregir cliente/fecha/notas/líneas); una vez
// confirmada, solo admite transiciones de estado (Cancelar, generar factura).
export function SaleEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const confirmDialog = useConfirm();
  const toast = useToast();
  const canManage = useCanManage();
  const isNew = id === "new";

  const [sale, setSale] = useState<Sale | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [ranges, setRanges] = useState<NumberingRange[]>([]);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [customerId, setCustomerId] = useState("");
  const [issueDate, setIssueDate] = useState(today());
  const [dueDate, setDueDate] = useState("");
  const [notes, setNotes] = useState("");
  const [lines, setLines] = useState<SalesLineInput[]>([]);
  const [paymentMeans, setPaymentMeans] = useState<PaymentMean[]>([]);
  const [rangeId, setRangeId] = useState("");
  const [linkedInvoiceStatus, setLinkedInvoiceStatus] = useState<DocumentStatus | null>(null);

  const { data: paymentTerms } = useCatalog(listPaymentTerms);
  const { data: paymentMethods } = useCatalog(listPaymentMethods);

  useEffect(() => {
    listCustomers().then(setCustomers).catch(() => setCustomers([]));
  }, []);

  useEffect(() => {
    if (isNew || !id) return;
    setLoading(true);
    fetchSale(id)
      .then((s) => {
        setSale(s);
        if (s.status === "draft") {
          setCustomerId(s.customer_id);
          setIssueDate(s.issue_date);
          setDueDate(s.due_date ?? "");
          setNotes(s.notes);
          setLines(s.lines);
          setPaymentMeans(s.payment_means ?? []);
        }
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la venta"))
      .finally(() => setLoading(false));
  }, [id, isNew]);

  // Editable mientras es nueva O sigue en borrador -- una vez confirmada, sales/ no soporta
  // editar (solo Cancelar/transiciones de estado).
  const isEditable = isNew || sale?.status === "draft";

  // Rangos activos de Factura Electrónica — solo hacen falta si la venta ya está confirmada
  // (para "Generar factura electrónica"), no tiene sentido cargarlos antes.
  useEffect(() => {
    if (sale?.status !== "confirmed") return;
    listNumberingRanges(INVOICE_DIAN_DOCUMENT_TYPE)
      .then((all) => setRanges(all.filter((r) => r.status === "active")))
      .catch(() => setRanges([]));
  }, [sale?.status]);

  // Si ya se generó una factura desde esta venta, hay que saber su estado real: si quedó
  // rechazada (u otro estado terminal recuperable), se vuelve a mostrar el selector de rango para
  // poder corregir y reintentar en vez de dejar la venta apuntando para siempre a una factura
  // rechazada sin salida (ver backend isRetryableFailure).
  useEffect(() => {
    if (!sale?.invoice_document_id) {
      setLinkedInvoiceStatus(null);
      return;
    }
    getDocument(sale.invoice_document_id)
      .then((doc) => setLinkedInvoiceStatus(doc.status))
      .catch(() => setLinkedInvoiceStatus(null));
  }, [sale?.invoice_document_id]);

  const invoiceIsRetryable = linkedInvoiceStatus !== null && RETRYABLE_INVOICE_STATUSES.includes(linkedInvoiceStatus);

  const customerOptions = customers.map((c) => ({ value: c.id, label: c.name }));
  const customerName = (cid: string) => customers.find((c) => c.id === cid)?.name ?? "—";

  async function handleCreate() {
    setError(null);
    setSaving(true);
    try {
      const created = await createSale({
        customer_id: customerId, issue_date: issueDate,
        due_date: dueDate || undefined, notes, lines, payment_means: paymentMeans,
      });
      toast.success(`Venta ${created.number || ""} creada.`);
      navigate(`/sales/${created.id}`, { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo crear la venta");
    } finally {
      setSaving(false);
    }
  }

  async function handleConfirm() {
    if (!sale) return;
    if (!(await confirmDialog("¿Confirmar esta venta? Se descontará del inventario y se contabilizará automáticamente — no podrás editarla después.", {}))) return;
    setSaving(true);
    try {
      setSale(await confirmSale(sale.id));
      toast.success("Venta confirmada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo confirmar la venta");
    } finally {
      setSaving(false);
    }
  }

  async function handleCancel() {
    if (!sale) return;
    if (!(await confirmDialog("¿Cancelar esta venta? Esta acción no se puede deshacer.", { tone: "danger" }))) return;
    setSaving(true);
    try {
      await cancelSale(sale.id);
      setSale({ ...sale, status: "cancelled" });
      toast.success("Venta cancelada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo cancelar la venta");
    } finally {
      setSaving(false);
    }
  }

  async function handleSaveEdit() {
    if (!sale) return;
    setError(null);
    setSaving(true);
    try {
      const updated = await updateSale(sale.id, {
        customer_id: customerId, issue_date: issueDate,
        due_date: dueDate || undefined, notes, lines, payment_means: paymentMeans,
      });
      setSale(updated);
      toast.success("Venta actualizada.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo actualizar la venta");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!sale) return;
    if (!(await confirmDialog(`¿Eliminar la venta ${sale.number || "en borrador"}? Esta acción no se puede deshacer.`, { tone: "danger" }))) return;
    try {
      await deleteSale(sale.id);
      toast.success("Venta eliminada.");
      navigate("/sales/orders");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar la venta");
    }
  }

  async function handleGenerateInvoice() {
    if (!sale || !rangeId) return;
    setGenerating(true);
    try {
      const doc = await createInvoiceFromSale(sale.id, rangeId);
      toast.success("Borrador de factura creado desde la venta.");
      navigate(`/documents/invoices/${doc.id}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo generar la factura");
    } finally {
      setGenerating(false);
    }
  }

  if (loading) {
    return <div className="flex min-h-32 items-center justify-center p-4"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>;
  }

  const title = isNew ? "Nueva venta" : sale ? `Venta ${sale.number || "(borrador)"}` : "Venta";

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Ventas", to: "/sales" }, sale ? { label: sale.number || "(borrador)", muted: true } : { label: title }]} />
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <ShoppingCart className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          {title}
        </h1>
        <div className="flex flex-wrap items-center gap-1.5">
          {sale && <StatusPill tone={STATUS_TONE[sale.status]} label={STATUS_LABEL[sale.status]} />}
          {sale?.status === "draft" && (
            <Button type="button" variant="danger" icon={<Trash2 className="h-3.5 w-3.5" />} onClick={handleDelete}>
              Eliminar
            </Button>
          )}
          {canManage && (sale?.status === "draft" || sale?.status === "confirmed") && (
            <Button type="button" variant="danger" icon={<Ban className="h-3.5 w-3.5" />} loading={saving} onClick={handleCancel}>
              Cancelar
            </Button>
          )}
          {sale?.status === "draft" && (
            <Button type="button" variant="success" icon={<Check className="h-3.5 w-3.5" />} loading={saving} onClick={handleConfirm}>
              Confirmar
            </Button>
          )}
        </div>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      <Card className="flex flex-col gap-4 p-4">
        {/* Datos del documento */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          {isEditable ? (
            <>
              <Input label="Fecha" type="date" value={issueDate} onChange={(e) => setIssueDate(e.target.value)} />
              <Input label="Vence cartera (opcional)" type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
              <div className="sm:col-span-3">
                <Input label="Notas" value={notes} onChange={(e) => setNotes(e.target.value)} />
              </div>
            </>
          ) : sale ? (
            <>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Fecha</p>
                <p className="text-xs text-(--text-primary)">{formatDateOnly(sale.issue_date)}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Vence cartera</p>
                <p className="text-xs text-(--text-primary)">{sale.due_date ? formatDateOnly(sale.due_date) : "—"}</p>
              </div>
              {sale.notes && (
                <div className="sm:col-span-3">
                  <p className="text-xs font-medium text-(--text-secondary)">Notas</p>
                  <p className="text-xs text-(--text-primary)">{sale.notes}</p>
                </div>
              )}
            </>
          ) : null}
        </div>

        {/* Cliente */}
        <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
          <h2 className="text-xs font-semibold text-(--text-primary)">Cliente</h2>
          {isEditable ? (
            <Combobox label="Cliente" value={customerId} onChange={setCustomerId} options={customerOptions} placeholder="Buscar cliente…" />
          ) : sale ? (
            <p className="text-xs text-(--text-primary)">{customerName(sale.customer_id)}</p>
          ) : null}
        </section>

        {/* Líneas */}
        <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
          <h2 className="text-xs font-semibold text-(--text-primary)">Líneas</h2>
          <SalesLineItemsEditor lines={isEditable ? lines : sale?.lines ?? []} onChange={setLines} disabled={!isEditable} />
        </section>

        {/* Forma de pago */}
        <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
          <h2 className="text-xs font-semibold text-(--text-primary)">Forma de pago</h2>
          {isEditable ? (
            <PaymentMeansEditor paymentMeans={paymentMeans} onChange={setPaymentMeans} />
          ) : sale?.payment_means && sale.payment_means.length > 0 ? (
            <p className="text-xs text-(--text-primary)">
              {sale.payment_means
                .map((pm) => {
                  const term = paymentTerms.find((t) => t.code === pm.code)?.name ?? pm.code;
                  const method = paymentMethods.find((m) => m.code === pm.payment_method_code)?.name ?? pm.payment_method_code;
                  return `${term} — ${method}`;
                })
                .join(", ")}
            </p>
          ) : (
            <p className="text-xs text-(--text-muted)">—</p>
          )}
        </section>

        {/* Totales */}
        <section className="grid grid-cols-12 gap-3 border-t border-(--border-color) pt-3">
          <div className="col-span-12 sm:col-span-4 sm:col-start-9">
            <SalesTotalsSummary lines={isEditable ? lines : sale?.lines ?? []} />
          </div>
        </section>

        {isEditable && (
          <div className="flex gap-2">
            <Button type="button" variant="secondary" onClick={() => navigate("/sales")} className="flex-1">
              Cancelar
            </Button>
            <Button
              type="button"
              disabled={!customerId || lines.length === 0}
              loading={saving}
              onClick={isNew ? handleCreate : handleSaveEdit}
              className="flex-1"
            >
              {isNew ? "Crear venta" : "Guardar cambios"}
            </Button>
          </div>
        )}
      </Card>

      {sale?.status === "confirmed" && sale.invoice_document_id && !invoiceIsRetryable && (
        <Card className="mt-3 p-4">
          <h3 className="mb-2 flex items-center gap-2 text-xs font-semibold text-(--text-primary)">
            <FileText className="h-3.5 w-3.5 text-(--color-success)" />
            Ya se generó la factura electrónica de esta venta
          </h3>
          <Button type="button" variant="secondary" icon={<FileText className="h-3.5 w-3.5" />} onClick={() => navigate(`/documents/invoices/${sale.invoice_document_id}`)}>
            Ver factura
          </Button>
        </Card>
      )}

      {sale?.status === "confirmed" && (!sale.invoice_document_id || invoiceIsRetryable) && (
        <Card className="mt-3 p-4">
          <h3 className="mb-3 flex items-center gap-2 text-xs font-semibold text-(--text-primary)">
            <FileText className="h-3.5 w-3.5 text-(--accent-primary)" />
            Generar factura electrónica
          </h3>
          {invoiceIsRetryable && (
            <div className="mb-3">
              <Banner tone="danger">
                La factura electrónica generada anteriormente fue rechazada por la DIAN y no consumió el consecutivo —{" "}
                <button type="button" className="underline" onClick={() => navigate(`/documents/invoices/${sale.invoice_document_id}`)}>
                  revisa el motivo del rechazo
                </button>
                , corrige lo necesario (cliente, líneas, forma de pago) y genera una factura nueva.
              </Banner>
            </div>
          )}
          {ranges.length === 0 ? (
            <p className="text-xs text-(--text-secondary)">
              No tienes un rango de numeración activo para Factura Electrónica — configúralo en{" "}
              <button type="button" className="text-(--accent-primary) hover:underline" onClick={() => navigate("/settings/company")}>
                Configuración › Empresa
              </button>.
            </p>
          ) : (
            <div className="flex flex-wrap items-end gap-3">
              <div className="w-56">
                <Combobox
                  label="Rango de numeración"
                  value={rangeId}
                  onChange={setRangeId}
                  options={ranges.map((r) => ({ value: r.id, label: `${r.prefix ? r.prefix + " " : ""}(${r.range_from}–${r.range_to ?? "∞"})` }))}
                  placeholder="Elegir rango…"
                />
              </div>
              <Button type="button" icon={<FileText className="h-3.5 w-3.5" />} disabled={!rangeId} loading={generating} onClick={handleGenerateInvoice}>
                Generar factura
              </Button>
            </div>
          )}
          <p className="mt-2 text-xs text-(--text-muted)">
            Crea un borrador de factura con las mismas líneas de esta venta — lo revisas y confirmas en Documentos, igual que cualquier otra.
          </p>
        </Card>
      )}
    </div>
  );
}
