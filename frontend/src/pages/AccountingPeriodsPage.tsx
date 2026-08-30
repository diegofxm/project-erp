import { useEffect, useState } from "react";
import { CalendarClock, Lock, LockOpen, Plus, Receipt, Pencil, Archive } from "lucide-react";
import { closePeriod, createVoucherType, deactivateVoucherType, listPeriods, listVoucherTypes, reopenPeriod, updateVoucherType } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { AccountingPeriod, VoucherType } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { InfoTip } from "../components/ui/InfoTip";
import { StatusPill } from "../components/ui/StatusPill";

const MONTHS = ["Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"];

export function AccountingPeriodsPage() {
  const [periods, setPeriods] = useState<AccountingPeriod[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [closingId, setClosingId] = useState<string | null>(null);
  const confirmDialog = useConfirm();
  const toast = useToast();

  const [voucherTypes, setVoucherTypes] = useState<VoucherType[]>([]);
  const [showVoucherForm, setShowVoucherForm] = useState(false);
  const [vtCode, setVtCode] = useState("");
  const [vtName, setVtName] = useState("");
  const [savingVt, setSavingVt] = useState(false);

  const [editingVt, setEditingVt] = useState<VoucherType | null>(null);
  const [editVtName, setEditVtName] = useState("");
  const [savingEditVt, setSavingEditVt] = useState(false);

  function refresh() {
    listPeriods()
      .then(setPeriods)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar los periodos"));
  }

  useEffect(() => {
    refresh();
    listVoucherTypes().then(setVoucherTypes).catch(() => setVoucherTypes([]));
  }, []);

  async function handleClose(p: AccountingPeriod) {
    if (!(await confirmDialog(
      `¿Cerrar ${MONTHS[p.month - 1]} ${p.year}? Ya no se podrán registrar ni anular asientos en este periodo.`,
      { tone: "danger" },
    ))) return;
    setClosingId(p.id);
    try {
      await closePeriod(p.id);
      toast.success(`${MONTHS[p.month - 1]} ${p.year} cerrado.`);
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo cerrar el periodo");
    } finally {
      setClosingId(null);
    }
  }

  async function handleReopen(p: AccountingPeriod) {
    if (!(await confirmDialog(
      `¿Reabrir ${MONTHS[p.month - 1]} ${p.year}? Es una acción excepcional — vuelve a permitir asientos y anulaciones sobre un mes ya cerrado.`,
      { tone: "danger" },
    ))) return;
    setClosingId(p.id);
    try {
      await reopenPeriod(p.id);
      toast.success(`${MONTHS[p.month - 1]} ${p.year} reabierto.`);
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo reabrir el periodo");
    } finally {
      setClosingId(null);
    }
  }

  async function handleCreateVoucherType() {
    if (!vtCode || !vtName) return;
    setSavingVt(true);
    try {
      await createVoucherType({ code: vtCode, name: vtName, resets_annually: true });
      toast.success("Tipo de comprobante creado.");
      setShowVoucherForm(false);
      setVtCode(""); setVtName("");
      listVoucherTypes().then(setVoucherTypes);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo crear el tipo de comprobante");
    } finally {
      setSavingVt(false);
    }
  }

  async function handleSaveEditVt() {
    if (!editingVt || !editVtName) return;
    setSavingEditVt(true);
    try {
      await updateVoucherType(editingVt.code, { name: editVtName, resets_annually: editingVt.resets_annually });
      toast.success("Tipo de comprobante actualizado.");
      setEditingVt(null);
      listVoucherTypes().then(setVoucherTypes);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo actualizar el tipo de comprobante");
    } finally {
      setSavingEditVt(false);
    }
  }

  async function handleDeactivateVt(v: VoucherType) {
    if (!(await confirmDialog(`¿Eliminar el tipo de comprobante "${v.code} — ${v.name}"?`, { tone: "danger" }))) return;
    try {
      await deactivateVoucherType(v.code);
      toast.success("Tipo de comprobante eliminado.");
      listVoucherTypes().then(setVoucherTypes);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el tipo de comprobante");
    }
  }

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Periodos" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <CalendarClock className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Periodos contables
        <InfoTip>
          Cada mes con actividad se abre automáticamente al registrar el primer asiento. Cerrar un periodo
          bloquea nuevos asientos y anulaciones sobre él — úsalo una vez concilies y cuadres el mes.
        </InfoTip>
      </h1>

      {error && <Banner tone="danger">{error}</Banner>}

      {periods === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : periods.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Todavía no hay periodos — se crean automáticamente con el primer asiento.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Periodo</th>
                <th className="px-3 py-2 font-medium">Estado</th>
                <th className="px-3 py-2 font-medium">Cerrado el</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {periods.map((p, i) => (
                <tr key={p.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 text-(--text-primary)">{MONTHS[p.month - 1]} {p.year}</td>
                  <td className="px-3 py-2">
                    <StatusPill tone={p.status === "OPEN" ? "success" : "neutral"} label={p.status === "OPEN" ? "Abierto" : "Cerrado"} />
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">{p.closed_at ? new Date(p.closed_at).toLocaleDateString("es-CO") : "—"}</td>
                  <td className="px-3 py-2">
                    {p.status === "OPEN" ? (
                      <button
                        type="button"
                        disabled={closingId === p.id}
                        onClick={() => handleClose(p)}
                        className="inline-flex items-center gap-1 rounded px-2 py-1 text-(--text-secondary) transition-colors hover:bg-(--bg-hover) hover:text-(--color-danger-text) disabled:opacity-60"
                      >
                        <Lock className="h-3 w-3" /> Cerrar
                      </button>
                    ) : (
                      <button
                        type="button"
                        disabled={closingId === p.id}
                        onClick={() => handleReopen(p)}
                        className="inline-flex items-center gap-1 rounded px-2 py-1 text-(--text-secondary) transition-colors hover:bg-(--bg-hover) hover:text-(--color-warning-text) disabled:opacity-60"
                      >
                        <LockOpen className="h-3 w-3" /> Reabrir
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-6 mb-2 flex items-center justify-between">
        <h2 className="flex items-center gap-2 text-xs font-semibold text-(--text-primary)">
          <Receipt className="h-3.5 w-3.5 text-(--accent-primary)" />
          Tipos de comprobante
        </h2>
        <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowVoucherForm((v) => !v)}>
          {showVoucherForm ? "Cancelar" : "Nuevo tipo"}
        </Button>
      </div>
      {showVoucherForm && (
        <Card className="mb-3 p-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <Input label="Código" required value={vtCode} onChange={(e) => setVtCode(e.target.value)} placeholder="Ej. CN" />
            <div className="sm:col-span-2">
              <Input label="Nombre" required value={vtName} onChange={(e) => setVtName(e.target.value)} placeholder="Ej. Comprobante de nómina" />
            </div>
          </div>
          <div className="mt-3 flex justify-end">
            <Button type="button" disabled={!vtCode || !vtName} loading={savingVt} onClick={handleCreateVoucherType}>Crear</Button>
          </div>
        </Card>
      )}
      {voucherTypes.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">
          Sin tipos de comprobante personalizados — el sistema ya usa CE/CI/NC/NI/CJ/AP internamente para los asientos automáticos.
        </p>
      ) : (
        <div className="flex flex-wrap gap-1.5">
          {voucherTypes.map((v) => (
            <span key={v.id} className="flex items-center gap-1 rounded bg-(--bg-tertiary) py-1 pl-2 pr-1 text-[11px] text-(--text-secondary)">
              {v.code} — {v.name}
              <button type="button" title="Editar" aria-label={`Editar ${v.code}`} onClick={() => { setEditingVt(v); setEditVtName(v.name); }}
                className="rounded p-0.5 text-(--text-muted) transition-colors hover:bg-(--bg-hover) hover:text-(--text-primary)">
                <Pencil className="h-3 w-3" />
              </button>
              <button type="button" title="Eliminar" aria-label={`Eliminar ${v.code}`} onClick={() => handleDeactivateVt(v)}
                className="rounded p-0.5 text-(--color-danger) transition-colors hover:bg-(--bg-hover)">
                <Archive className="h-3 w-3" />
              </button>
            </span>
          ))}
        </div>
      )}

      {editingVt && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setEditingVt(null)}>
          <Card className="w-full max-w-sm p-4" onClick={(e) => e.stopPropagation()}>
            <h3 className="mb-3 text-xs font-semibold text-(--text-primary)">Editar tipo de comprobante ({editingVt.code})</h3>
            <Input label="Nombre" required value={editVtName} onChange={(e) => setEditVtName(e.target.value)} />
            <div className="mt-4 flex justify-end gap-2">
              <Button type="button" variant="secondary" onClick={() => setEditingVt(null)}>Cancelar</Button>
              <Button type="button" loading={savingEditVt} disabled={!editVtName} onClick={handleSaveEditVt}>Guardar</Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}
