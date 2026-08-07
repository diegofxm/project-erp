import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { FileBarChart } from "lucide-react";
import { getAccountLedger, getBSReport, getPLReport, getTrialBalance, listAccounts } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { formatDateOnly, todayColombiaISO } from "../lib/dateFormat";
import type { Account, AccountBalance, LedgerLine, TrialBalanceRow } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Combobox } from "../components/ui/Combobox";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";

type ReportType = "pl" | "bs" | "trial" | "ledger";

const REPORT_LABEL: Record<ReportType, string> = {
  pl: "Estado de Resultados", bs: "Balance General", trial: "Balance de Prueba", ledger: "Libro Mayor",
};

function money(cents: number): string {
  return formatCOP.format(cents / 100);
}

function todayISO(): string {
  return todayColombiaISO();
}

function firstOfYearISO(): string {
  return `${new Date().getFullYear()}-01-01`;
}

const REPORT_TYPES = Object.keys(REPORT_LABEL) as ReportType[];

export function AccountingReportsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialType = searchParams.get("type") as ReportType | null;
  const [report, setReportState] = useState<ReportType>(initialType && REPORT_TYPES.includes(initialType) ? initialType : "pl");
  function setReport(r: ReportType) {
    setReportState(r);
    setSearchParams((prev) => { prev.set("type", r); return prev; }, { replace: true });
  }
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const [year, setYear] = useState(new Date().getFullYear());
  const [asOf, setAsOf] = useState(todayISO());
  const [from, setFrom] = useState(firstOfYearISO());
  const [to, setTo] = useState(todayISO());
  const [accountCode, setAccountCode] = useState("");

  const [accounts, setAccounts] = useState<Account[]>([]);
  const [plRows, setPlRows] = useState<AccountBalance[] | null>(null);
  const [bsRows, setBsRows] = useState<AccountBalance[] | null>(null);
  const [trialRows, setTrialRows] = useState<TrialBalanceRow[] | null>(null);
  const [ledgerRows, setLedgerRows] = useState<LedgerLine[] | null>(null);

  useEffect(() => {
    listAccounts().then(setAccounts).catch(() => setAccounts([]));
  }, []);

  const accountOptions = useMemo(
    () => accounts.filter((a) => a.is_posting).map((a) => ({ value: a.code, label: `${a.code} — ${a.name}` })),
    [accounts],
  );

  function run() {
    setError(null);
    setLoading(true);
    const onErr = (err: unknown) => setError(err instanceof ApiError ? err.message : "No se pudo generar el reporte");
    if (report === "pl") {
      getPLReport(year).then(setPlRows).catch(onErr).finally(() => setLoading(false));
    } else if (report === "bs") {
      getBSReport(asOf).then(setBsRows).catch(onErr).finally(() => setLoading(false));
    } else if (report === "trial") {
      getTrialBalance(from, to).then(setTrialRows).catch(onErr).finally(() => setLoading(false));
    } else {
      if (!accountCode) { setLoading(false); return; }
      getAccountLedger(accountCode, from, to).then(setLedgerRows).catch(onErr).finally(() => setLoading(false));
    }
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { run(); }, [report]);

  const plGroups = useMemo(() => {
    if (!plRows) return null;
    const income = plRows.filter((r) => r.category === "Ingreso");
    const expenses = plRows.filter((r) => r.category !== "Ingreso");
    const totalIncome = income.reduce((s, r) => s + -r.balance, 0); // Ingreso es cuenta crédito: balance negativo = saldo real positivo
    const totalExpenses = expenses.reduce((s, r) => s + r.balance, 0);
    return { income, expenses, totalIncome, totalExpenses, net: totalIncome - totalExpenses };
  }, [plRows]);

  const bsGroups = useMemo(() => {
    if (!bsRows) return null;
    const assets = bsRows.filter((r) => r.category === "Activo");
    const liabilities = bsRows.filter((r) => r.category === "Pasivo");
    const equity = bsRows.filter((r) => r.category === "Patrimonio");
    const totalAssets = assets.reduce((s, r) => s + r.balance, 0);
    const totalLiabilities = liabilities.reduce((s, r) => s + -r.balance, 0);
    const totalEquity = equity.reduce((s, r) => s + -r.balance, 0);
    return { assets, liabilities, equity, totalAssets, totalLiabilities, totalEquity };
  }, [bsRows]);

  const trialTotals = useMemo(() => {
    if (!trialRows) return null;
    return trialRows.reduce((acc, r) => ({ debit: acc.debit + r.debit, credit: acc.credit + r.credit }), { debit: 0, credit: 0 });
  }, [trialRows]);

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Reportes" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <FileBarChart className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Reportes financieros
      </h1>

      <div className="mb-3 flex h-9 w-fit overflow-hidden rounded border border-(--border-color)">
        {(Object.keys(REPORT_LABEL) as ReportType[]).map((r) => (
          <button
            key={r}
            type="button"
            onClick={() => setReport(r)}
            className={`px-3 text-xs font-medium transition-colors ${
              report === r ? "bg-(--accent-primary) text-white" : "bg-(--bg-secondary) text-(--text-secondary) hover:bg-(--bg-hover)"
            }`}
          >
            {REPORT_LABEL[r]}
          </button>
        ))}
      </div>

      <div className="mb-3 flex flex-wrap items-end gap-2">
        {report === "pl" && (
          <Input type="number" label="Año" value={year} onChange={(e) => setYear(Number(e.target.value))} className="w-28" />
        )}
        {report === "bs" && (
          <Input type="date" label="Corte al" value={asOf} onChange={(e) => setAsOf(e.target.value)} />
        )}
        {(report === "trial" || report === "ledger") && (
          <>
            <Input type="date" label="Desde" value={from} onChange={(e) => setFrom(e.target.value)} />
            <Input type="date" label="Hasta" value={to} onChange={(e) => setTo(e.target.value)} />
          </>
        )}
        {report === "ledger" && (
          <div className="w-64">
            <Combobox label="Cuenta" value={accountCode} onChange={setAccountCode} options={accountOptions} placeholder="Buscar cuenta..." />
          </div>
        )}
        <button
          type="button"
          onClick={run}
          className="h-[34px] rounded bg-(--accent-primary) px-3 text-xs font-medium text-white transition-colors hover:bg-(--accent-hover) sm:h-[30px]"
        >
          Generar
        </button>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {loading ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : (
        <>
          {report === "pl" && plGroups && (
            plGroups.income.length === 0 && plGroups.expenses.length === 0 ? (
              <p className="text-xs text-(--text-secondary)">Sin movimientos en {year}.</p>
            ) : (
              <div className="overflow-x-auto rounded border border-(--border-color)">
                <table className="w-full text-left text-xs">
                  <tbody>
                    <tr className="bg-(--bg-tertiary)"><td className="px-3 py-2 font-semibold text-(--text-primary)" colSpan={2}>Ingresos</td></tr>
                    {plGroups.income.map((r) => (
                      <tr key={r.account_id}><td className="px-3 py-1.5 pl-6 text-(--text-secondary)">{r.account_code} — {r.account_name}</td><td className="px-3 py-1.5 text-right font-mono text-(--text-primary)">{money(-r.balance)}</td></tr>
                    ))}
                    <tr className="border-t border-(--border-light) font-medium"><td className="px-3 py-1.5">Total ingresos</td><td className="px-3 py-1.5 text-right font-mono">{money(plGroups.totalIncome)}</td></tr>

                    <tr className="bg-(--bg-tertiary)"><td className="px-3 py-2 font-semibold text-(--text-primary)" colSpan={2}>Gastos y costos</td></tr>
                    {plGroups.expenses.map((r) => (
                      <tr key={r.account_id}><td className="px-3 py-1.5 pl-6 text-(--text-secondary)">{r.account_code} — {r.account_name}</td><td className="px-3 py-1.5 text-right font-mono text-(--text-primary)">{money(r.balance)}</td></tr>
                    ))}
                    <tr className="border-t border-(--border-light) font-medium"><td className="px-3 py-1.5">Total gastos y costos</td><td className="px-3 py-1.5 text-right font-mono">{money(plGroups.totalExpenses)}</td></tr>

                    <tr className={`border-t-2 border-(--border-color) font-semibold ${plGroups.net >= 0 ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>
                      <td className="px-3 py-2">Utilidad neta</td><td className="px-3 py-2 text-right font-mono">{money(plGroups.net)}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            )
          )}

          {report === "bs" && bsGroups && (
            bsGroups.assets.length === 0 && bsGroups.liabilities.length === 0 && bsGroups.equity.length === 0 ? (
              <p className="text-xs text-(--text-secondary)">Sin saldos al {formatDateOnly(asOf)}.</p>
            ) : (
              <div className="overflow-x-auto rounded border border-(--border-color)">
                <table className="w-full text-left text-xs">
                  <tbody>
                    <tr className="bg-(--bg-tertiary)"><td className="px-3 py-2 font-semibold text-(--text-primary)" colSpan={2}>Activo</td></tr>
                    {bsGroups.assets.map((r) => (
                      <tr key={r.account_id}><td className="px-3 py-1.5 pl-6 text-(--text-secondary)">{r.account_code} — {r.account_name}</td><td className="px-3 py-1.5 text-right font-mono text-(--text-primary)">{money(r.balance)}</td></tr>
                    ))}
                    <tr className="border-t border-(--border-light) font-medium"><td className="px-3 py-1.5">Total activo</td><td className="px-3 py-1.5 text-right font-mono">{money(bsGroups.totalAssets)}</td></tr>

                    <tr className="bg-(--bg-tertiary)"><td className="px-3 py-2 font-semibold text-(--text-primary)" colSpan={2}>Pasivo</td></tr>
                    {bsGroups.liabilities.map((r) => (
                      <tr key={r.account_id}><td className="px-3 py-1.5 pl-6 text-(--text-secondary)">{r.account_code} — {r.account_name}</td><td className="px-3 py-1.5 text-right font-mono text-(--text-primary)">{money(-r.balance)}</td></tr>
                    ))}
                    <tr className="border-t border-(--border-light) font-medium"><td className="px-3 py-1.5">Total pasivo</td><td className="px-3 py-1.5 text-right font-mono">{money(bsGroups.totalLiabilities)}</td></tr>

                    <tr className="bg-(--bg-tertiary)"><td className="px-3 py-2 font-semibold text-(--text-primary)" colSpan={2}>Patrimonio</td></tr>
                    {bsGroups.equity.map((r) => (
                      <tr key={r.account_id}><td className="px-3 py-1.5 pl-6 text-(--text-secondary)">{r.account_code} — {r.account_name}</td><td className="px-3 py-1.5 text-right font-mono text-(--text-primary)">{money(-r.balance)}</td></tr>
                    ))}
                    <tr className="border-t border-(--border-light) font-medium"><td className="px-3 py-1.5">Total patrimonio</td><td className="px-3 py-1.5 text-right font-mono">{money(bsGroups.totalEquity)}</td></tr>

                    <tr className="border-t-2 border-(--border-color) font-semibold"><td className="px-3 py-2">Pasivo + Patrimonio</td><td className="px-3 py-2 text-right font-mono">{money(bsGroups.totalLiabilities + bsGroups.totalEquity)}</td></tr>
                  </tbody>
                </table>
              </div>
            )
          )}

          {report === "trial" && trialRows && (
            trialRows.length === 0 ? (
              <p className="text-xs text-(--text-secondary)">Sin movimientos en el rango seleccionado.</p>
            ) : (
              <div className="overflow-x-auto rounded border border-(--border-color)">
                <table className="w-full text-left text-xs">
                  <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                    <tr>
                      <th className="px-3 py-2 font-medium">Cuenta</th>
                      <th className="px-3 py-2 font-medium">Débito</th>
                      <th className="px-3 py-2 font-medium">Crédito</th>
                      <th className="px-3 py-2 font-medium">Saldo</th>
                    </tr>
                  </thead>
                  <tbody>
                    {trialRows.map((r, i) => (
                      <tr key={r.account_id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                        <td className="px-3 py-1.5 text-(--text-primary)">{r.account_code} — {r.account_name}</td>
                        <td className="px-3 py-1.5 font-mono text-(--text-primary)">{money(r.debit)}</td>
                        <td className="px-3 py-1.5 font-mono text-(--text-primary)">{money(r.credit)}</td>
                        <td className="px-3 py-1.5 font-mono text-(--text-primary)">{money(r.balance)}</td>
                      </tr>
                    ))}
                  </tbody>
                  {trialTotals && (
                    <tfoot>
                      <tr className="border-t-2 border-(--border-color) font-semibold">
                        <td className="px-3 py-2">Total</td>
                        <td className="px-3 py-2 font-mono">{money(trialTotals.debit)}</td>
                        <td className="px-3 py-2 font-mono">{money(trialTotals.credit)}</td>
                        <td className="px-3 py-2" />
                      </tr>
                    </tfoot>
                  )}
                </table>
              </div>
            )
          )}

          {report === "ledger" && (
            !accountCode ? (
              <p className="text-xs text-(--text-secondary)">Elige una cuenta para ver su movimiento.</p>
            ) : ledgerRows && ledgerRows.length === 0 ? (
              <p className="text-xs text-(--text-secondary)">Sin movimientos para esta cuenta en el rango seleccionado.</p>
            ) : ledgerRows ? (
              <div className="overflow-x-auto rounded border border-(--border-color)">
                <table className="w-full text-left text-xs">
                  <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                    <tr>
                      <th className="px-3 py-2 font-medium">Fecha</th>
                      <th className="px-3 py-2 font-medium">Comprobante</th>
                      <th className="px-3 py-2 font-medium">Descripción</th>
                      <th className="px-3 py-2 font-medium">Débito</th>
                      <th className="px-3 py-2 font-medium">Crédito</th>
                      <th className="px-3 py-2 font-medium">Saldo</th>
                    </tr>
                  </thead>
                  <tbody>
                    {ledgerRows.map((l, i) => (
                      <tr key={`${l.journal_id}-${i}`} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                        <td className="px-3 py-1.5 text-(--text-secondary)">{formatDateOnly(l.date)}</td>
                        <td className="px-3 py-1.5 font-mono text-(--text-secondary)">{l.voucher_number || "—"}</td>
                        <td className="px-3 py-1.5 text-(--text-primary)">{l.description}</td>
                        <td className="px-3 py-1.5 font-mono text-(--text-primary)">{l.debit ? money(l.debit) : "—"}</td>
                        <td className="px-3 py-1.5 font-mono text-(--text-primary)">{l.credit ? money(l.credit) : "—"}</td>
                        <td className="px-3 py-1.5 font-mono text-(--text-primary)">{money(l.running_balance)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : null
          )}
        </>
      )}
    </div>
  );
}
