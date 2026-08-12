import { useEffect, useState } from "react";
import { ShoppingBag, Check, Ban, PackageCheck, Trash2, FileText, Mail, Receipt, Plus } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import { addWithholding, cancelPurchase, confirmPurchase, createPurchase, deletePurchase, fetchPurchase, getPurchasePdfBlobUrl, receivePurchase, sendPurchaseEmail, updatePurchase } from "../lib/purchases";
import { listSuppliers } from "../lib/suppliers";
import { listNumberingRanges } from "../lib/numberingRanges";
import { listWithholdingConcepts } from "../lib/accounting";
import { createSupportDocFromPurchase } from "../lib/documents";
import { listPaymentMethods, listPaymentTerms } from "../lib/catalogs";
import { useCatalog } from "../lib/useCatalog";
import { ApiError } from "../lib/apiClient";
import { openInNewTab } from "../lib/openInNewTab";
import { useConfirm } from "../context/ConfirmContext";
import { useCanManage } from "../hooks/useCanManage";
import { useToast } from "../context/ToastContext";
import { formatCOP } from "../lib/currency";
import { formatDateOnly, todayColombiaISO } from "../lib/dateFormat";
import type { Purchase, PurchaseStatus, PurchaseLineInput, NumberingRange, PaymentMean, Supplier, WithholdingConcept } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Combobox } from "../components/ui/Combobox";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { SendEmailModal } from "../components/ui/SendEmailModal";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";
import { SalesLineItemsEditor } from "../components/sales/SalesLineItemsEditor";
import { SalesTotalsSummary } from "../components/sales/SalesTotalsSummary";
import { PaymentMeansEditor } from "../components/invoice-form/PaymentMeansEditor";

const STATUS_LABEL: Record<PurchaseStatus, string> = {
  draft: "Borrador", confirmed: "Confirmada", received: "Recibida", cancelled: "Cancelada",
};
const STATUS_TONE: Record<PurchaseStatus, StatusTone> = {
  draft: "neutral", confirmed: "info", received: "success", cancelled: "danger",
};
const SUPPORT_DOC_DIAN_TYPE = "05";
const today = () => todayColombiaISO();

// Editable mientras está en borrador (crear o corregir proveedor/fecha/notas/líneas). A
// diferencia de una venta, el ciclo tiene un paso extra: Confirmar (compromiso con el proveedor)
// y luego Recibir (entra a inventario y se contabiliza) son pasos separados.
export function PurchaseOrderEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const confirmDialog = useConfirm();
  const toast = useToast();
  const canManage = useCanManage();
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
  const [issueDate, setIssueDate] = useState(today());
  const [dueDate, setDueDate] = useState("");
  const [notes, setNotes] = useState("");
  const [lines, setLines] = useState<PurchaseLineInput[]>([]);
  const [paymentMeans, setPaymentMeans] = useState<PaymentMean[]>([]);
  const [rangeId, setRangeId] = useState("");
  const [concepts, setConcepts] = useState<WithholdingConcept[]>([]);
  const [whConceptId, setWhConceptId] = useState("");
  const [whBase, setWhBase] = useState("");
  const [addingWh, setAddingWh] = useState(false);

  const { data: paymentTerms } = useCatalog(listPaymentTerms);
  const { data: paymentMethods } = useCatalog(listPaymentMethods);

  useEffect(() => {
    listSuppliers().then(setSuppliers).catch(() => setSuppliers([]));
    listWithholdingConcepts().then(setConcepts).catch(() => setConcepts([]));
  }, []);

  useEffect(() => {
    if (isNew || !id) return;
    setLoading(true);
    fetchPurchase(id)
      .then((p) => {
        setPurchase(p);
        if (p.status === "draft") {
          setSupplierId(p.supplier_id);
          setIssueDate(p.issue_date);
          setDueDate(p.due_date ?? "");
          setNotes(p.notes);
          setLines(p.lines);
          setPaymentMeans(p.payment_means ?? []);
        }
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar la orden de compra"))
      .finally(() => setLoading(false));
  }, [id, isNew]);

  const isEditable = isNew || purchase?.status === "draft";

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
        supplier_id: supplierId, issue_date: issueDate,
        due_date: dueDate || undefined, notes, lines, payment_means: paymentMeans,
      });
      toast.success(`Orden de compra ${created.number || ""} creada.`);
      navigate(`/purchases/${created.id}`, { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo crear la orden de compra");
    } finally {
      setSaving(false);
    }
  }

  async function handleSaveEdit() {
    if (!purchase) return;
    setError(null);
    setSaving(true);
    try {
      const updated = await updatePurchase(purchase.id, {
        supplier_id: supplierId, issue_date: issueDate,
        due_date: dueDate || undefined, notes, lines, payment_means: paymentMeans,
      });
      setPurchase(updated);
      toast.success("Orden de compra actualizada.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo actualizar la orden de compra");
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

  async function handleAddWithholding() {
    if (!purchase || !whConceptId || !whBase) return;
    setAddingWh(true);
    try {
      await addWithholding(purchase.id, whConceptId, Number(whBase));
      const updated = await fetchPurchase(purchase.id);
      setPurchase(updated);
      setWhConceptId("");
      setWhBase("");
      toast.success("Retención aplicada.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo aplicar la retención");
    } finally {
      setAddingWh(false);
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
  const canSendEmail = purchase?.status === "confirmed" || purchase?.status === "received";

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Compras", to: "/purchases" }, purchase ? { label: purchase.number || "(borrador)", muted: true } : { label: title }]} />
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <ShoppingBag className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          {title}
        </h1>
        <div className="flex flex-wrap items-center gap-1.5">
          {purchase && <StatusPill tone={STATUS_TONE[purchase.status]} label={STATUS_LABEL[purchase.status]} />}
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
          {canManage && (purchase?.status === "draft" || purchase?.status === "confirmed") && (
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

      <Card className="flex flex-col gap-4 p-4">
        {/* Datos del documento */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          {isEditable ? (
            <>
              <Input label="Fecha" type="date" value={issueDate} onChange={(e) => setIssueDate(e.target.value)} />
              <Input label="Recepción esperada (opcional)" type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
              <div className="sm:col-span-3">
                <Input label="Notas" value={notes} onChange={(e) => setNotes(e.target.value)} />
              </div>
            </>
          ) : purchase ? (
            <>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Fecha</p>
                <p className="text-xs text-(--text-primary)">{formatDateOnly(purchase.issue_date)}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-(--text-secondary)">Recepción esperada</p>
                <p className="text-xs text-(--text-primary)">{purchase.due_date ? formatDateOnly(purchase.due_date) : "—"}</p>
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

        {/* Proveedor */}
        <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
          <h2 className="text-xs font-semibold text-(--text-primary)">Proveedor</h2>
          {isEditable ? (
            <Combobox label="Proveedor" value={supplierId} onChange={setSupplierId} options={supplierOptions} placeholder="Buscar proveedor…" />
          ) : purchase ? (
            <p className="text-xs text-(--text-primary)">{supplierName(purchase.supplier_id)}</p>
          ) : null}
        </section>

        {/* Líneas */}
        <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
          <h2 className="text-xs font-semibold text-(--text-primary)">Líneas</h2>
          <SalesLineItemsEditor lines={isEditable ? lines : purchase?.lines ?? []} onChange={setLines} disabled={!isEditable} />
        </section>

        {/* Forma de pago */}
        <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
          <h2 className="text-xs font-semibold text-(--text-primary)">Forma de pago</h2>
          {isEditable ? (
            <PaymentMeansEditor paymentMeans={paymentMeans} onChange={setPaymentMeans} />
          ) : purchase?.payment_means && purchase.payment_means.length > 0 ? (
            <p className="text-xs text-(--text-primary)">
              {purchase.payment_means
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
            <SalesTotalsSummary lines={isEditable ? lines : purchase?.lines ?? []} />
          </div>
        </section>

        {isEditable && (
          <div className="flex gap-2">
            <Button type="button" variant="secondary" onClick={() => navigate("/purchases")} className="flex-1">
              Cancelar
            </Button>
            <Button
              type="button"
              disabled={!supplierId || lines.length === 0}
              loading={saving}
              onClick={isNew ? handleCreate : handleSaveEdit}
              className="flex-1"
            >
              {isNew ? "Crear orden" : "Guardar cambios"}
            </Button>
          </div>
        )}
      </Card>

      {purchase && (purchase.status === "confirmed" || purchase.status === "received") && (
        <Card className="mt-3 p-4">
          <h3 className="mb-3 flex items-center gap-2 text-xs font-semibold text-(--text-primary)">
            <Receipt className="h-3.5 w-3.5 text-(--accent-primary)" />
            Retenciones
          </h3>
          {purchase.withholdings.length > 0 && (
            <div className="mb-3 overflow-x-auto rounded border border-(--border-color)">
              <table className="w-full text-left text-xs">
                <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                  <tr>
                    <th className="px-3 py-2 font-medium">Concepto</th>
                    <th className="px-3 py-2 font-medium">Base</th>
                    <th className="px-3 py-2 font-medium">Tarifa</th>
                    <th className="px-3 py-2 font-medium">Monto</th>
                  </tr>
                </thead>
                <tbody>
                  {purchase.withholdings.map((w, i) => (
                    <tr key={w.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                      <td className="px-3 py-2 text-(--text-primary)">{w.concept_name}</td>
                      <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(w.base)}</td>
                      <td className="px-3 py-2 text-(--text-secondary)">{(w.rate_bp / 100).toFixed(2)}%</td>
                      <td className="px-3 py-2 font-mono text-(--text-primary)">{formatCOP.format(w.amount)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {purchase.status === "confirmed" ? (
            <div className="flex flex-wrap items-end gap-2">
              <div className="w-64">
                <Combobox
                  label="Concepto"
                  value={whConceptId}
                  onChange={setWhConceptId}
                  options={concepts.map((c) => ({ value: c.id, label: `${c.code} — ${c.name} (${(c.rate_bp / 100).toFixed(2)}%)` }))}
                  placeholder="Buscar concepto…"
                />
              </div>
              <Input label="Base" type="number" min="0" value={whBase} onChange={(e) => setWhBase(e.target.value)} className="w-36" />
              <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} disabled={!whConceptId || !whBase} loading={addingWh} onClick={handleAddWithholding}>
                Aplicar
              </Button>
            </div>
          ) : purchase.withholdings.length === 0 ? (
            <p className="text-xs text-(--text-secondary)">Esta orden se recibió sin retenciones aplicadas.</p>
          ) : null}
        </Card>
      )}

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
