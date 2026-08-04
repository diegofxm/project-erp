import { useEffect, useState } from "react";
import { ShoppingBag, Check, Ban, PackageCheck, Trash2, FileText, Mail } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import { cancelPurchase, confirmPurchase, createPurchase, deletePurchase, fetchPurchase, getPurchasePdfBlobUrl, receivePurchase, sendPurchaseEmail } from "../lib/purchases";
import { listSuppliers } from "../lib/suppliers";
import { listNumberingRanges } from "../lib/numberingRanges";
import { createSupportDocFromPurchase } from "../lib/documents";
import { ApiError } from "../lib/apiClient";
import { openInNewTab } from "../lib/openInNewTab";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import { formatCOP } from "../lib/currency";
import type { Purchase, PurchaseStatus, PurchaseLineInput, NumberingRange, Supplier } from "../lib/types";
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

const STATUS_LABEL: Record<PurchaseStatus, string> = {
  draft: "Borrador", confirmed: "Confirmada", received: "Recibida", cancelled: "Cancelada",
};
const STATUS_TONE: Record<PurchaseStatus, StatusTone> = {
  draft: "neutral", confirmed: "info", received: "success", cancelled: "danger",
};
const SUPPORT_DOC_DIAN_TYPE = "05";
const today = () => new Date().toISOString().slice(0, 10);

// Igual que SaleEditorPage: purchase/ no tiene endpoint de actualización (solo Create/Confirm/
// Receive/Cancel/Delete), así que una orden ya creada es de solo lectura salvo transiciones de
// estado. A diferencia de una venta, el ciclo tiene un paso extra: Confirmar (compromiso con el
// proveedor) y luego Recibir (entra a inventario y se contabiliza) son pasos separados.
export function PurchaseOrderEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const confirmDialog = useConfirm();
  const toast = useToast();
  const isNew = id === "new";

  const [purchase, setPurchase] = useState<Purchase | null>(null);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [ranges, setRanges] = useState<NumberingRange[]>([]);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [loadingPdf, setLoadingPdf] = useState(false);
  const [showEmailModal, setShowEmailModal] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [supplierId, setSupplierId] = useState("");
  const [number, setNumber] = useState("");
  const [issueDate, setIssueDate] = useState(today());
  const [dueDate, setDueDate] = useState("");
  const [notes, setNotes] = useState("");
  const [lines, setLines] = useState<PurchaseLineInput[]>([]);
  const [rangeId, setRangeId] = useState("");

  useEffect(() => {
    listSuppliers().then(setSuppliers).catch(() => setSuppliers([]));
  }, []);

  useEffect(() => {
    if (isNew || !id) return;
    setLoading(true);
    fetchPurchase(id)
      .then(setPurchase)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la orden de compra"))
      .finally(() => setLoading(false));
  }, [id, isNew]);

  // Rangos activos de Documento Soporte — solo hacen falta si la orden ya está recibida (para
  // "Generar Documento Soporte"), no tiene sentido cargarlos antes.
  useEffect(() => {
    if (purchase?.status !== "received") return;
    listNumberingRanges(SUPPORT_DOC_DIAN_TYPE)
      .then((all) => setRanges(all.filter((r) => r.status === "active")))
      .catch(() => setRanges([]));
  }, [purchase?.status]);

  const supplierOptions = suppliers.map((s) => ({ value: s.id, label: s.name }));
  const supplierName = (sid: string) => suppliers.find((s) => s.id === sid)?.name ?? "—";

  async function handleCreate() {
    setError(null);
    setSaving(true);
    try {
      const created = await createPurchase({
        supplier_id: supplierId, number, issue_date: issueDate,
        due_date: dueDate || undefined, notes, lines,
      });
      toast.success(`Orden de compra ${created.number || ""} creada.`);
      navigate(`/purchases/${created.id}`, { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo crear la orden de compra");
    } finally {
      setSaving(false);
    }
  }

  async function handleConfirm() {
    if (!purchase) return;
    if (!(await confirmDialog("¿Confirmar esta orden de compra? Es el compromiso formal con el proveedor.", {}))) return;
    setSaving(true);
    try {
      setPurchase(await confirmPurchase(purchase.id));
      toast.success("Orden de compra confirmada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo confirmar la orden");
    } finally {
      setSaving(false);
    }
  }

  async function handleReceive() {
    if (!purchase) return;
    if (!(await confirmDialog("¿Marcar como recibida? Aumentará el inventario y se contabilizará automáticamente.", {}))) return;
    setSaving(true);
    try {
      setPurchase(await receivePurchase(purchase.id));
      toast.success("Mercancía recibida.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo marcar como recibida");
    } finally {
      setSaving(false);
    }
  }

  async function handleCancel() {
    if (!purchase) return;
    if (!(await confirmDialog("¿Cancelar esta orden de compra? Esta acción no se puede deshacer.", { tone: "danger" }))) return;
    setSaving(true);
    try {
      await cancelPurchase(purchase.id);
      setPurchase({ ...purchase, status: "cancelled" });
      toast.success("Orden de compra cancelada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo cancelar la orden");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!purchase) return;
    if (!(await confirmDialog(`¿Eliminar la orden ${purchase.number || "en borrador"}? Esta acción no se puede deshacer.`, { tone: "danger" }))) return;
    try {
      await deletePurchase(purchase.id);
      toast.success("Orden de compra eliminada.");
      navigate("/purchases/orders");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar la orden");
    }
  }

  async function handleGenerateSupportDoc() {
    if (!purchase || !rangeId) return;
    setGenerating(true);
    try {
      const doc = await createSupportDocFromPurchase(purchase.id, rangeId);
      toast.success("Borrador de Documento Soporte creado desde la orden.");
      navigate(`/documents/support-documents/${doc.id}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo generar el Documento Soporte");
    } finally {
      setGenerating(false);
    }
  }

  async function handleViewPdf() {
    if (!purchase) return;
    setLoadingPdf(true);
    try {
      const url = await getPurchasePdfBlobUrl(purchase.id);
      openInNewTab(url);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo generar el PDF");
    } finally {
      setLoadingPdf(false);
    }
  }

  async function handleSendEmailConfirm(to: string, _cc: string[]) {
    if (!purchase) return;
    try {
      await sendPurchaseEmail(purchase.id, to);
      toast.success(`Orden de compra enviada a ${to}.`);
      setShowEmailModal(false);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo enviar la orden de compra");
      throw err;
    }
  }

  if (loading) {
    return <div className="flex min-h-32 items-center justify-center p-4"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>;
  }

  const title = isNew ? "Nueva orden de compra" : purchase ? `Orden ${purchase.number || "(borrador)"}` : "Orden de compra";
  const total = isNew ? salesLinesTotal(lines) : purchase ? salesLinesTotal(purchase.lines) : 0;
  const canSendEmail = purchase?.status === "confirmed" || purchase?.status === "received";

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Compras", to: "/purchases" }, { label: title }]} />
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <ShoppingBag className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          {title}
        </h1>
        <div className="flex flex-wrap items-center gap-1.5">
          {purchase && <StatusPill tone={STATUS_TONE[purchase.status]} label={STATUS_LABEL[purchase.status]} />}
          {isNew && (
            <Button type="button" loading={saving} disabled={!supplierId || !number || lines.length === 0} onClick={handleCreate}>
              Guardar orden
            </Button>
          )}
          {!isNew && purchase && (
            <Button type="button" variant="secondary" icon={<FileText className="h-3.5 w-3.5" />} loading={loadingPdf} onClick={handleViewPdf}>
              Ver PDF
            </Button>
          )}
          {canSendEmail && (
            <Button type="button" variant="secondary" icon={<Mail className="h-3.5 w-3.5" />} onClick={() => setShowEmailModal(true)}>
              Enviar al proveedor
            </Button>
          )}
          {purchase?.status === "draft" && (
            <Button type="button" variant="danger" icon={<Trash2 className="h-3.5 w-3.5" />} onClick={handleDelete}>
              Eliminar
            </Button>
          )}
          {(purchase?.status === "draft" || purchase?.status === "confirmed") && (
            <Button type="button" variant="danger" icon={<Ban className="h-3.5 w-3.5" />} loading={saving} onClick={handleCancel}>
              Cancelar
            </Button>
          )}
          {purchase?.status === "draft" && (
            <Button type="button" variant="success" icon={<Check className="h-3.5 w-3.5" />} loading={saving} onClick={handleConfirm}>
              Confirmar
            </Button>
          )}
          {purchase?.status === "confirmed" && (
            <Button type="button" variant="success" icon={<PackageCheck className="h-3.5 w-3.5" />} loading={saving} onClick={handleReceive}>
              Marcar recibida
            </Button>
          )}
        </div>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      <Card className="p-4">
        <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
          {isNew ? (
            <>
              <Combobox label="Proveedor" value={supplierId} onChange={setSupplierId} options={supplierOptions} placeholder="Buscar proveedor…" />
              <Input label="Número interno" required value={number} onChange={(e) => setNumber(e.target.value)} placeholder="Ej. OC-001" />
              <Input label="Fecha" type="date" value={issueDate} onChange={(e) => setIssueDate(e.target.value)} />
              <Input label="Recepción esperada (opcional)" type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
              <Input label="Notas" value={notes} onChange={(e) => setNotes(e.target.value)} />
            </>
          ) : purchase ? (
            <>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Proveedor</p>
                <p className="text-xs text-(--text-primary)">{supplierName(purchase.supplier_id)}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Fecha</p>
                <p className="text-xs text-(--text-primary)">{new Date(purchase.issue_date).toLocaleDateString("es-CO")}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Recepción esperada</p>
                <p className="text-xs text-(--text-primary)">{purchase.due_date ? new Date(purchase.due_date).toLocaleDateString("es-CO") : "—"}</p>
              </div>
              {purchase.notes && (
                <div className="sm:col-span-3">
                  <p className="text-xs font-medium text-(--text-secondary)">Notas</p>
                  <p className="text-xs text-(--text-primary)">{purchase.notes}</p>
                </div>
              )}
            </>
          ) : null}
        </div>

        <SalesLineItemsEditor lines={isNew ? lines : purchase?.lines ?? []} onChange={setLines} disabled={!isNew} />

        <div className="mt-4 flex justify-end border-t border-(--border-light) pt-3">
          <span className="text-xs text-(--text-secondary)">
            Total: <span className="font-mono text-sm font-semibold text-(--text-primary)">{formatCOP.format(total)}</span>
          </span>
        </div>
      </Card>

      {purchase?.status === "received" && purchase.support_document_id && (
        <Card className="mt-3 p-4">
          <h3 className="mb-2 flex items-center gap-2 text-xs font-semibold text-(--text-primary)">
            <FileText className="h-3.5 w-3.5 text-(--color-success)" />
            Ya se generó el Documento Soporte de esta orden
          </h3>
          <Button type="button" variant="secondary" icon={<FileText className="h-3.5 w-3.5" />} onClick={() => navigate(`/documents/support-documents/${purchase.support_document_id}`)}>
            Ver Documento Soporte
          </Button>
        </Card>
      )}

      {purchase?.status === "received" && !purchase.support_document_id && (
        <Card className="mt-3 p-4">
          <h3 className="mb-3 flex items-center gap-2 text-xs font-semibold text-(--text-primary)">
            <FileText className="h-3.5 w-3.5 text-(--accent-primary)" />
            Generar Documento Soporte
          </h3>
          {ranges.length === 0 ? (
            <p className="text-xs text-(--text-secondary)">
              No tienes un rango de numeración activo para Documento Soporte — configúralo en{" "}
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
              <Button type="button" icon={<FileText className="h-3.5 w-3.5" />} disabled={!rangeId} loading={generating} onClick={handleGenerateSupportDoc}>
                Generar Documento Soporte
              </Button>
            </div>
          )}
          <p className="mt-2 text-xs text-(--text-muted)">
            Crea un borrador de Documento Soporte con las mismas líneas de esta orden — úsalo cuando el proveedor no está obligado a facturar electrónicamente. Lo revisas y confirmas en Documentos, igual que cualquier otro.
          </p>
        </Card>
      )}

      {showEmailModal && purchase && (
        <SendEmailModal
          initialEmail={suppliers.find((s) => s.id === purchase.supplier_id)?.email ?? ""}
          onSend={handleSendEmailConfirm}
          onClose={() => setShowEmailModal(false)}
        />
      )}
    </div>
  );
}
