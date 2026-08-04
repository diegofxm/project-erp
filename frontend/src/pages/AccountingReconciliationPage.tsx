import { useEffect, useMemo, useState } from "react";
import { GitMerge } from "lucide-react";
import { listAccounts, listOpenLines, markReconciled } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { formatDateOnly } from "../lib/dateFormat";
import { useToast } from "../context/ToastContext";
import type { Account, OpenLine } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Combobox } from "../components/ui/Combobox";
import { InfoTip } from "../components/ui/InfoTip";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";

function money(cents: number): string {
  return formatCOP.format(cents / 100);
}

export function AccountingReconciliationPage() {
  const toast = useToast();
  const [pucAccounts, setPucAccounts] = useState<Account[]>([]);
  const [accountCode, setAccountCode] = useState<string | null>(null);
  const [lines, setLines] = useState<OpenLine[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string[]>([]);
  const [marking, setMarking] = useState(false);

  useEffect(() => {
    listAccounts().then(setPucAccounts).catch(() => setPucAccounts([]));
  }, []);

  const accountOptions = useMemo(
    () => pucAccounts.filter((a) => a.is_posting).map((a) => ({ value: a.code, label: `${a.code} — ${a.name}` })),
    [pucAccounts],
  );

  function refresh(code: string) {
    setLines(null);
    listOpenLines(code)
      .then(setLines)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar las líneas sin conciliar"));
  }

  useEffect(() => {
    if (!accountCode) { setLines(null); return; }
    setSelected([]);
    refresh(accountCode);
  }, [accountCode]);

  function toggle(lineId: string) {
    setSelected((prev) => {
      if (prev.includes(lineId)) return prev.filter((id) => id !== lineId);
      if (prev.length >= 2) return [prev[1], lineId];
      return [...prev, lineId];
    });
  }

  async function handleCross() {
    if (selected.length !== 2 || !accountCode) return;
    setMarking(true);
    try {
      await markReconciled(selected[0], selected[1], "");
      toast.success("Líneas cruzadas.");
      setSelected([]);
      refresh(accountCode);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo cruzar las líneas");
    } finally {
      setMarking(false);
    }
  }

  const selectedTotal = lines
    ? lines.filter((l) => selected.includes(l.line_id)).reduce((sum, l) => sum + l.debit_cents - l.credit_cents, 0)
    : 0;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Conciliación de cuentas" }]} />
      <h1 className="mb-1 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <GitMerge className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Conciliación de cuentas
        <InfoTip>Cruza dos líneas de asiento de la misma cuenta que se cancelan entre sí (ej. el débito de una factura de cartera contra el crédito de su pago) — distinto de la conciliación bancaria, que cruza contra un extracto.</InfoTip>
      </h1>
      <p className="mb-3 text-xs text-(--text-secondary)">Elige una cuenta y selecciona dos líneas sin conciliar para cruzarlas entre sí.</p>

      {error && <Banner tone="danger">{error}</Banner>}

      <div className="mb-3 max-w-sm">
        <Combobox label="Cuenta" value={accountCode ?? ""} onChange={setAccountCode} options={accountOptions} placeholder="Buscar cuenta…" />
      </div>

      {!accountCode ? (
        <p className="text-xs text-(--text-secondary)">Selecciona una cuenta para ver sus líneas sin conciliar.</p>
      ) : lines === null ? (
        <div className="flex min-h-32 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
      ) : lines.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Sin líneas pendientes de conciliar en esta cuenta.</p>
      ) : (
        <>
          <div className="overflow-x-auto rounded border border-(--border-color)">
            <table className="w-full text-left text-xs">
              <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                <tr>
                  <th className="px-3 py-2" />
                  <th className="px-3 py-2 font-medium">Fecha</th>
                  <th className="px-3 py-2 font-medium">Comprobante</th>
                  <th className="px-3 py-2 font-medium">Descripción</th>
                  <th className="px-3 py-2 font-medium">NIT tercero</th>
                  <th className="px-3 py-2 font-medium">Débito</th>
                  <th className="px-3 py-2 font-medium">Crédito</th>
                </tr>
              </thead>
              <tbody>
                {lines.map((l, i) => (
                  <tr
                    key={l.line_id}
                    onClick={() => toggle(l.line_id)}
                    className={`cursor-pointer ${selected.includes(l.line_id) ? "bg-(--bg-selected)" : i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"} hover:bg-(--bg-hover)`}
                  >
                    <td className="px-3 py-2">
                      <input type="checkbox" checked={selected.includes(l.line_id)} onChange={() => toggle(l.line_id)} onClick={(e) => e.stopPropagation()} />
                    </td>
                    <td className="px-3 py-2 text-(--text-secondary)">{formatDateOnly(l.date)}</td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{l.voucher_number || "—"}</td>
                    <td className="px-3 py-2 text-(--text-primary)">{l.description}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{l.third_party_nit || "—"}</td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{l.debit_cents ? money(l.debit_cents) : "—"}</td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{l.credit_cents ? money(l.credit_cents) : "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {selected.length > 0 && (
            <div className="mt-3 flex items-center justify-between rounded border border-(--border-color) bg-(--bg-secondary) px-3 py-2">
              <span className="text-xs text-(--text-secondary)">
                {selected.length} línea{selected.length > 1 ? "s" : ""} seleccionada{selected.length > 1 ? "s" : ""} · diferencia {money(selectedTotal)}
              </span>
              <Button type="button" loading={marking} disabled={selected.length !== 2} onClick={handleCross}>
                Cruzar seleccionadas
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
