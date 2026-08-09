import { useEffect, useState } from "react";
import { ShoppingCart, Check, Ban, FileText } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import { cancelSale, confirmSale, createSale, fetchSale } from "../lib/sales";
import { listCustomers } from "../lib/customers";
import { listNumberingRanges } from "../lib/numberingRanges";
import { createInvoiceFromSale } from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { useConfirm } from "../context/ConfirmContext";
import { useCanManage } from "../hooks/useCanManage";
import { useToast } from "../context/ToastContext";
import { formatCOP } from "../lib/currency";
import { formatDateOnly, todayColombiaISO } from "../lib/dateFormat";
import type { Customer, NumberingRange, Sale, SaleStatus, SalesLineInput } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Combobox } from "../components/ui/Combobox";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";
import { SalesLineItemsEditor, salesLinesTotal } from "../components/sales/SalesLineItemsEditor";

const STATUS_LABEL: Record<SaleStatus, string> = { draft: "Borrador", confirmed: "Confirmada", cancelled: "Cancelada" };
const STATUS_TONE: Record<SaleStatus, StatusTone> = { draft: "neutral", confirmed: "success", cancelled: "danger" };
const INVOICE_DIAN_DOCUMENT_TYPE = "01";
const today = () => todayColombiaISO();

// Igual que QuoteEditorPage: sales/ no tiene endpoint de actualización (solo Create/Confirm/
// Cancel), así que una venta ya creada es de solo lectura salvo sus transiciones de estado.
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
  const [number, setNumber] = useState("");
  const [issueDate, setIssueDate] = useState(today());
  const [dueDate, setDueDate] = useState("");
  const [notes, setNotes] = useState("");
  const [lines, setLines] = useState<SalesLineInput[]>([]);
  const [rangeId, setRangeId] = useState("");

  useEffect(() => {
    listCustomers().then(setCustomers).catch(() => setCustomers([]));
  }, []);

  useEffect(() => {
    if (isNew || !id) return;
    setLoading(true);
    fetchSale(id)
      .then(setSale)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la venta"))
      .finally(() => setLoading(false));
  }, [id, isNew]);

  // Rangos activos de Factura Electrónica — solo hacen falta si la venta ya está confirmada
  // (para "Generar factura electrónica"), no tiene sentido cargarlos antes.
  useEffect(() => {
    if (sale?.status !== "confirmed") return;
    listNumberingRanges(INVOICE_DIAN_DOCUMENT_TYPE)
      .then((all) => setRanges(all.filter((r) => r.status === "active")))
      .catch(() => setRanges([]));
  }, [sale?.status]);

  const customerOptions = customers.map((c) => ({ value: c.id, label: c.name }));
  const customerName = (cid: string) => customers.find((c) => c.id === cid)?.name ?? "—";

  async function handleCreate() {
    setError(null);
    setSaving(true);
    try {
      const created = await createSale({
        customer_id: customerId, number, issue_date: issueDate,
        due_date: dueDate || undefined, notes, lines,
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
  const total = isNew ? salesLinesTotal(lines) : sale ? salesLinesTotal(sale.lines) : 0;

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
          {isNew && (
            <Button type="button" loading={saving} disabled={!customerId || !number || lines.length === 0} onClick={handleCreate}>
              Guardar venta
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

      <Card className="p-4">
        <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
          {isNew ? (
            <>
              <Combobox label="Cliente" value={customerId} onChange={setCustomerId} options={customerOptions} placeholder="Buscar cliente…" />
              <Input label="Número interno" required value={number} onChange={(e) => setNumber(e.target.value)} placeholder="Ej. V-001" />
              <Input label="Fecha" type="date" value={issueDate} onChange={(e) => setIssueDate(e.target.value)} />
              <Input label="Vence cartera (opcional)" type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
              <Input label="Notas" value={notes} onChange={(e) => setNotes(e.target.value)} />
            </>
          ) : sale ? (
            <>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Cliente</p>
                <p className="text-xs text-(--text-primary)">{customerName(sale.customer_id)}</p>
              </div>
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

        <SalesLineItemsEditor lines={isNew ? lines : sale?.lines ?? []} onChange={setLines} disabled={!isNew} />

        <div className="mt-4 flex justify-end border-t border-(--border-light) pt-3">
          <span className="text-xs text-(--text-secondary)">
            Total: <span className="font-mono text-sm font-semibold text-(--text-primary)">{formatCOP.format(total)}</span>
          </span>
        </div>
      </Card>

      {sale?.status === "confirmed" && sale.invoice_document_id && (
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

      {sale?.status === "confirmed" && !sale.invoice_document_id && (
        <Card className="mt-3 p-4">
          <h3 className="mb-3 flex items-center gap-2 text-xs font-semibold text-(--text-primary)">
            <FileText className="h-3.5 w-3.5 text-(--accent-primary)" />
            Generar factura electrónica
          </h3>
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
