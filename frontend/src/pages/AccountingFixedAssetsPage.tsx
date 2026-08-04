import { useEffect, useMemo, useState } from "react";
import { Boxes, Plus, PlayCircle } from "lucide-react";
import { createFixedAsset, listAccounts, listFixedAssets, runDepreciation } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { useToast } from "../context/ToastContext";
import type { Account, FixedAsset } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Combobox } from "../components/ui/Combobox";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { InfoTip } from "../components/ui/InfoTip";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

function money(v: number): string {
  return formatCOP.format(v / 100);
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

const STATUS_LABEL: Record<string, string> = { ACTIVE: "Activo", FULLY_DEPRECIATED: "Depreciado totalmente", DISPOSED: "Dado de baja" };
const STATUS_TONE: Record<string, StatusTone> = { ACTIVE: "success", FULLY_DEPRECIATED: "neutral", DISPOSED: "danger" };

const EMPTY = {
  code: "", name: "", description: "", asset_account: "", depreciation_account: "", accumulated_account: "",
  acquisition_date: todayISO(), acquisition_cost: "", salvage_value: "0", useful_life_months: "60",
};

export function AccountingFixedAssetsPage() {
  const toast = useToast();
  const [assets, setAssets] = useState<FixedAsset[] | null>(null);
  const [pucAccounts, setPucAccounts] = useState<Account[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState(EMPTY);
  const [saving, setSaving] = useState(false);
  const [runDate, setRunDate] = useState(todayISO());
  const [running, setRunning] = useState(false);

  function refresh() {
    listFixedAssets().then(setAssets).catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar los activos fijos"));
  }

  useEffect(() => {
    refresh();
    listAccounts().then(setPucAccounts).catch(() => setPucAccounts([]));
  }, []);

  const accountOptions = useMemo(
    () => pucAccounts.filter((a) => a.is_posting).map((a) => ({ value: a.code, label: `${a.code} — ${a.name}` })),
    [pucAccounts],
  );

  async function handleCreate() {
    setSaving(true);
    try {
      await createFixedAsset({
        code: form.code, name: form.name, description: form.description,
        asset_account: form.asset_account, depreciation_account: form.depreciation_account, accumulated_account: form.accumulated_account,
        acquisition_date: form.acquisition_date,
        acquisition_cost_cents: Math.round(Number(form.acquisition_cost) * 100),
        salvage_value_cents: Math.round(Number(form.salvage_value || 0) * 100),
        useful_life_months: Number(form.useful_life_months),
      });
      toast.success("Activo fijo creado.");
      setShowForm(false);
      setForm(EMPTY);
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo crear el activo");
    } finally {
      setSaving(false);
    }
  }

  async function handleRunDepreciation() {
    setRunning(true);
    try {
      await runDepreciation(runDate);
      toast.success("Depreciación del período corrida.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo correr la depreciación");
    } finally {
      setRunning(false);
    }
  }

  const canSave = form.code && form.name && form.asset_account && form.depreciation_account && form.accumulated_account && form.acquisition_cost;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Activos fijos" }]} />
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Boxes className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Activos fijos y depreciación
          <InfoTip>
            Depreciación por <strong>línea recta</strong>: (costo − valor residual) / vida útil en meses. La corrida
            mensual genera un único asiento con una línea de gasto y una de depreciación acumulada por cada activo con
            saldo pendiente — no se puede correr dos veces el mismo período.
          </InfoTip>
        </h1>
        <div className="flex flex-wrap items-center gap-1.5">
          <Input type="date" value={runDate} onChange={(e) => setRunDate(e.target.value)} className="w-40" />
          <Button type="button" variant="secondary" icon={<PlayCircle className="h-3.5 w-3.5" />} loading={running} onClick={handleRunDepreciation}>
            Correr depreciación
          </Button>
          <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowForm(true)}>
            Nuevo activo
          </Button>
        </div>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {showForm && (
        <Card className="mb-3 p-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <Input label="Código" required value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} placeholder="Ej. AF-001" />
            <div className="sm:col-span-2">
              <Input label="Nombre" required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Ej. Computador portátil" />
            </div>
            <Input label="Fecha de adquisición" type="date" value={form.acquisition_date} onChange={(e) => setForm({ ...form, acquisition_date: e.target.value })} />
            <Input label="Costo de adquisición" type="number" min="0" required value={form.acquisition_cost} onChange={(e) => setForm({ ...form, acquisition_cost: e.target.value })} />
            <Input label="Valor residual" type="number" min="0" value={form.salvage_value} onChange={(e) => setForm({ ...form, salvage_value: e.target.value })} />
            <Input label="Vida útil (meses)" type="number" min="1" value={form.useful_life_months} onChange={(e) => setForm({ ...form, useful_life_months: e.target.value })} />
            <Combobox label="Cuenta del activo" value={form.asset_account} onChange={(v) => setForm({ ...form, asset_account: v })} options={accountOptions} placeholder="Ej. 152405" />
            <Combobox label="Cuenta de gasto (depreciación)" value={form.depreciation_account} onChange={(v) => setForm({ ...form, depreciation_account: v })} options={accountOptions} placeholder="Ej. 516010" />
            <Combobox label="Cuenta de depreciación acumulada" value={form.accumulated_account} onChange={(v) => setForm({ ...form, accumulated_account: v })} options={accountOptions} placeholder="Ej. 159220" />
          </div>
          <div className="mt-3 flex justify-end gap-2 border-t border-(--border-light) pt-3">
            <Button type="button" variant="secondary" onClick={() => setShowForm(false)}>Cancelar</Button>
            <Button type="button" disabled={!canSave} loading={saving} onClick={handleCreate}>Crear activo</Button>
          </div>
        </Card>
      )}

      {assets === null ? (
        <div className="flex min-h-32 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
      ) : assets.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Todavía no tienes activos fijos registrados.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Código</th>
                <th className="px-3 py-2 font-medium">Nombre</th>
                <th className="px-3 py-2 font-medium">Costo</th>
                <th className="px-3 py-2 font-medium">Depreciación mensual</th>
                <th className="px-3 py-2 font-medium">Acumulada</th>
                <th className="px-3 py-2 font-medium">Estado</th>
              </tr>
            </thead>
            <tbody>
              {assets.map((a, i) => (
                <tr key={a.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 font-mono text-(--text-primary)">{a.code}</td>
                  <td className="px-3 py-2 text-(--text-primary)">{a.name}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{money(a.acquisition_cost)}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{money(a.monthly_depreciation)}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{money(a.accumulated)}</td>
                  <td className="px-3 py-2"><StatusPill tone={STATUS_TONE[a.status]} label={STATUS_LABEL[a.status]} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
