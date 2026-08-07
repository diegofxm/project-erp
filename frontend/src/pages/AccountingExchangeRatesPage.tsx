import { useEffect, useState } from "react";
import { DollarSign, Plus, RefreshCw, Search, X } from "lucide-react";
import { listExchangeRates, lookupExchangeRate, setExchangeRate, syncExchangeRate } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { addDaysColombiaISO, formatDateOnly, todayColombiaISO } from "../lib/dateFormat";
import { useToast } from "../context/ToastContext";
import type { ExchangeRate } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";

function todayISO(): string {
  return todayColombiaISO();
}

function daysAgoISO(days: number): string {
  return addDaysColombiaISO(-days);
}

// LookupTrmModal es solo informativo: consulta la TRM de una fecha puntual y la muestra en
// pantalla — el backend ya se encarga de mirar primero la base de datos y solo tocar la fuente
// oficial si esa fecha nunca se había pedido (ver ExchangeRateUseCase.GetOrFetch), así que este
// modal nunca "abusa" del servicio externo por sí mismo, sin importar cuántas veces se abra.
function LookupTrmModal({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const toast = useToast();
  const [date, setDate] = useState(todayISO());
  const [looking, setLooking] = useState(false);
  const [result, setResult] = useState<ExchangeRate | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleSearch() {
    if (!date) return;
    setLooking(true);
    setError(null);
    setResult(null);
    try {
      const r = await lookupExchangeRate(date);
      setResult(r);
      onSaved();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo obtener la TRM de esa fecha");
      toast.error(err instanceof ApiError ? err.message : "No se pudo obtener la TRM de esa fecha");
    } finally {
      setLooking(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div className="w-full max-w-sm rounded-lg border border-(--border-color) bg-(--bg-primary) p-5 shadow-xl" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Search className="h-4 w-4 text-(--accent-primary)" />
            <h2 className="text-sm font-semibold text-(--text-primary)">Buscar TRM de una fecha</h2>
          </div>
          <button type="button" onClick={onClose} className="text-(--text-muted) hover:text-(--text-primary) transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        {error && <Banner tone="danger">{error}</Banner>}

        <div className="flex items-end gap-2">
          <Input label="Fecha" type="date" max={todayISO()} value={date} onChange={(e) => setDate(e.target.value)} />
          <Button type="button" loading={looking} disabled={!date} onClick={handleSearch}>
            Buscar
          </Button>
        </div>

        {result && (
          <div className="mt-4 rounded border border-(--border-color) bg-(--bg-secondary) p-3">
            <p className="text-xs text-(--text-secondary)">TRM del {formatDateOnly(result.rate_date)}</p>
            <p className="text-lg font-semibold tabular-nums text-(--text-primary)">
              {result.rate.toLocaleString("es-CO", { minimumFractionDigits: 2, maximumFractionDigits: 4 })}
            </p>
            <p className="mt-1 text-xs text-(--text-secondary)">{result.description || "—"}</p>
          </div>
        )}

        <p className="mt-4 text-xs text-(--text-secondary)">
          Solo informativo — si esa fecha ya está en la base de datos se trae de ahí; si no, se busca una sola vez en la fuente oficial y se guarda para consultas futuras.
        </p>

        <div className="mt-4 flex justify-end">
          <Button type="button" variant="secondary" onClick={onClose}>Cerrar</Button>
        </div>
      </div>
    </div>
  );
}

export function AccountingExchangeRatesPage() {
  const toast = useToast();
  const [rates, setRates] = useState<ExchangeRate[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [showNew, setShowNew] = useState(false);
  const [showLookup, setShowLookup] = useState(false);
  const [date, setDate] = useState(todayISO());
  const [from, setFrom] = useState("USD");
  const [rate, setRate] = useState("");
  const [description, setDescription] = useState("");
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);

  const hasTodayRate = (rates ?? []).some((r) => r.rate_date === todayISO() && r.from_currency === "USD" && r.to_currency === "COP");

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
      await setExchangeRate({ rate_date: date, from_currency: from.toUpperCase(), rate: Number(rate), description: description || undefined });
      toast.success("Tasa de cambio guardada.");
      setShowNew(false);
      setRate("");
      setDescription("");
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
          <Button
            type="button"
            variant="secondary"
            loading={syncing}
            disabled={hasTodayRate}
            title={hasTodayRate ? "Ya se registró la TRM de hoy — se sincroniza sola todos los días a la 1:00 a.m." : undefined}
            icon={<RefreshCw className="h-3.5 w-3.5" />}
            onClick={handleSync}
          >
            {hasTodayRate ? "TRM de hoy ya sincronizada" : "Sincronizar TRM de hoy"}
          </Button>
          <Button type="button" variant="secondary" icon={<Search className="h-3.5 w-3.5" />} onClick={() => setShowLookup(true)}>
            Buscar fecha específica
          </Button>
          <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowNew(true)}>
            Registrar tasa
          </Button>
        </div>
      </div>

      <p className="mb-3 text-xs text-(--text-secondary)">
        La TRM se sincroniza sola todos los días a la 1:00 a.m. (hora Colombia) — este botón es solo un respaldo manual por si ese disparador llegara a fallar un día.
      </p>

      {error && <Banner tone="danger">{error}</Banner>}

      {showLookup && <LookupTrmModal onClose={() => setShowLookup(false)} onSaved={refresh} />}

      {showNew && (
        <Card className="mb-3 p-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-4">
            <Input label="Fecha" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            <Input label="Moneda origen" required value={from} onChange={(e) => setFrom(e.target.value)} placeholder="USD" maxLength={3} />
            <Input label="Destino" value="COP" disabled />
            <Input label="Tasa (pesos por unidad)" type="number" min="0" step="0.0001" required value={rate} onChange={(e) => setRate(e.target.value)} placeholder="4123.4567" />
          </div>
          <Input className="mt-3" label="Descripción (opcional)" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Editado manualmente" />
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
                  <td className="px-3 py-2 text-(--text-secondary)">{r.description || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
