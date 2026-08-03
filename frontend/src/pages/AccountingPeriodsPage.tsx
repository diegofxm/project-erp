import { useEffect, useState } from "react";
import { CalendarClock, Lock } from "lucide-react";
import { closePeriod, listPeriods } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { AccountingPeriod } from "../lib/types";
import { Banner } from "../components/ui/Banner";
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

  function refresh() {
    listPeriods()
      .then(setPeriods)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar los periodos"));
  }

  useEffect(() => { refresh(); }, []);

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
                    {p.status === "OPEN" && (
                      <button
                        type="button"
                        disabled={closingId === p.id}
                        onClick={() => handleClose(p)}
                        className="inline-flex items-center gap-1 rounded px-2 py-1 text-(--text-secondary) transition-colors hover:bg-(--bg-hover) hover:text-(--color-danger-text) disabled:opacity-60"
                      >
                        <Lock className="h-3 w-3" /> Cerrar
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
