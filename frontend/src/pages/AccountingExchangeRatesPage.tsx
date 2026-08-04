import { useEffect, useState } from "react";
import { DollarSign, Plus, RefreshCw } from "lucide-react";
import { listExchangeRates, setExchangeRate, syncExchangeRate } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { formatDateOnly } from "../lib/dateFormat";
import { useToast } from "../context/ToastContext";
import type { ExchangeRate } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

// sourceDescription explica de dónde salió la tasa — el texto de "dolarapi" es el mismo que esa
// fuente usa para describir la TRM en su propia página (Superintendencia Financiera).
function sourceDescription(source: string): string {
  switch (source) {
    case "DOLARAPI":
      return "TRM oficial de Colombia publicada por la Superintendencia Financiera";
    case "MANUAL":
      return "Editado manualmente";
    default:
      return "—";
  }
}

function daysAgoISO(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString().slice(0, 10);
}

export function AccountingExchangeRatesPage() {
  const toast = useToast();
  const [rates, setRates] = useState<ExchangeRate[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [showNew, setShowNew] = useState(false);
  const [date, setDate] = useState(todayISO());
  const [from, setFrom] = useState("USD");
  const [rate, setRate] = useState("");
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);

  function refresh() {
    listExchangeRates(daysAgoISO(90), todayISO())
      .then(setRates)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar las tasas de cambio"));
  }

  useEffect(() => { refresh(); }, []);

  async function handleSync() {
    setSyncing(true);
    try {
      const r = await syncExchangeRate();
      toast.success(`TRM sincronizada: ${r.rate.toLocaleString("es-CO")} (${formatDateOnly(r.rate_date)})`);
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo sincronizar la TRM — puedes registrarla manualmente mientras tanto");
    } finally {
      setSyncing(false);
    }
  }

  async function handleSave() {
    if (!from || !rate) return;
    setSaving(true);
    try {
      await setExchangeRate({ rate_date: date, from_currency: from.toUpperCase(), rate: Number(rate) });
      toast.success("Tasa de cambio guardada.");
      setShowNew(false);
      setRate("");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar la tasa de cambio");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Tasas de cambio" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <DollarSign className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Tasas de cambio (TRM)
        </h1>
        <div className="flex gap-2">
          <Button type="button" variant="secondary" loading={syncing} icon={<RefreshCw className="h-3.5 w-3.5" />} onClick={handleSync}>
            Sincronizar TRM de hoy
          </Button>
          <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowNew(true)}>
            Registrar tasa
          </Button>
        </div>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {showNew && (
        <Card className="mb-3 p-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-4">
            <Input label="Fecha" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            <Input label="Moneda origen" required value={from} onChange={(e) => setFrom(e.target.value)} placeholder="USD" maxLength={3} />
            <Input label="Destino" value="COP" disabled />
            <Input label="Tasa (pesos por unidad)" type="number" min="0" step="0.0001" required value={rate} onChange={(e) => setRate(e.target.value)} placeholder="4123.4567" />
          </div>
          <div className="mt-3 flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={() => setShowNew(false)}>Cancelar</Button>
            <Button type="button" loading={saving} disabled={!from || !rate} onClick={handleSave}>Guardar</Button>
          </div>
        </Card>
      )}

      {rates === null ? (
        <div className="flex min-h-32 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
      ) : rates.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Sin tasas de cambio registradas en los últimos 90 días.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Fecha</th>
                <th className="px-3 py-2 font-medium">Par</th>
                <th className="px-3 py-2 font-medium">Tasa</th>
                <th className="px-3 py-2 font-medium">Fuente</th>
                <th className="px-3 py-2 font-medium">Descripción</th>
              </tr>
            </thead>
            <tbody>
              {rates.map((r, i) => (
                <tr key={`${r.rate_date}-${r.from_currency}-${r.to_currency}`} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 text-(--text-secondary)">{formatDateOnly(r.rate_date)}</td>
                  <td className="px-3 py-2 font-mono text-(--text-primary)">{r.from_currency} → {r.to_currency}</td>
                  <td className="px-3 py-2 font-mono text-(--text-primary)">{r.rate.toLocaleString("es-CO", { minimumFractionDigits: 2, maximumFractionDigits: 4 })}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{r.source}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{sourceDescription(r.source)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
