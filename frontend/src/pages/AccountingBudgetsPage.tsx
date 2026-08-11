import { useEffect, useMemo, useState } from "react";
import { Calculator, Plus, Pencil, Trash2 } from "lucide-react";
import { approveBudget, compareBudget, createBudget, deleteBudget, deleteBudgetLine, listAccounts, listBudgetLines, listBudgets, setBudgetLine, updateBudget } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { Account, Budget, BudgetActualRow, BudgetLine } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Combobox } from "../components/ui/Combobox";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill } from "../components/ui/StatusPill";

const MONTHS = ["Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic"];

function money(v: number): string {
  return formatCOP.format(v / 100);
}

export function AccountingBudgetsPage() {
  const toast = useToast();
  const confirmDialog = useConfirm();
  const [budgets, setBudgets] = useState<Budget[] | null>(null);
  const [pucAccounts, setPucAccounts] = useState<Account[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [lines, setLines] = useState<BudgetLine[] | null>(null);
  const [compare, setCompare] = useState<BudgetActualRow[] | null>(null);
  const [showCompare, setShowCompare] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [showNew, setShowNew] = useState(false);
  const [newYear, setNewYear] = useState(new Date().getFullYear());
  const [newName, setNewName] = useState("");
  const [saving, setSaving] = useState(false);

  const [lineAccount, setLineAccount] = useState("");
  const [lineMonths, setLineMonths] = useState<string[]>(Array(12).fill(""));
  const [savingLine, setSavingLine] = useState(false);

  const [renaming, setRenaming] = useState(false);
  const [renameValue, setRenameValue] = useState("");
  const [savingRename, setSavingRename] = useState(false);

  function refreshBudgets() {
    listBudgets().then((list) => { setBudgets(list); if (!selected && list.length > 0) setSelected(list[0].id); })
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar los presupuestos"));
  }

  useEffect(() => {
    refreshBudgets();
    listAccounts().then(setPucAccounts).catch(() => setPucAccounts([]));
  }, []);

  function refreshLines() {
    if (!selected) return;
    listBudgetLines(selected).then(setLines).catch(() => setLines([]));
  }

  useEffect(() => { refreshLines(); setShowCompare(false); }, [selected]);

  const accountOptions = useMemo(
    () => pucAccounts.filter((a) => a.is_posting).map((a) => ({ value: a.code, label: `${a.code} — ${a.name}` })),
    [pucAccounts],
  );

  const selectedBudget = budgets?.find((b) => b.id === selected) ?? null;

  async function handleCreateBudget() {
    if (!newName) return;
    setSaving(true);
    try {
      const b = await createBudget(newYear, newName);
      toast.success("Presupuesto creado.");
      setShowNew(false);
      setNewName("");
      refreshBudgets();
      setSelected(b.id);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo crear el presupuesto");
    } finally {
      setSaving(false);
    }
  }

  async function handleSetLine() {
    if (!selected || !lineAccount) return;
    setSavingLine(true);
    try {
      const months = lineMonths.map((m) => Math.round(Number(m || 0) * 100));
      await setBudgetLine(selected, lineAccount, months);
      toast.success("Línea de presupuesto guardada.");
      setLineAccount("");
      setLineMonths(Array(12).fill(""));
      refreshLines();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar la línea");
    } finally {
      setSavingLine(false);
    }
  }

  async function handleRename() {
    if (!selected || !renameValue) return;
    setSavingRename(true);
    try {
      await updateBudget(selected, renameValue);
      toast.success("Presupuesto renombrado.");
      setRenaming(false);
      refreshBudgets();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo renombrar el presupuesto");
    } finally {
      setSavingRename(false);
    }
  }

  async function handleDeleteBudget() {
    if (!selectedBudget) return;
    if (!(await confirmDialog(`¿Eliminar el presupuesto "${selectedBudget.name}"? Esta acción no se puede deshacer.`, { tone: "danger" }))) return;
    try {
      await deleteBudget(selectedBudget.id);
      toast.success("Presupuesto eliminado.");
      setSelected(null);
      refreshBudgets();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el presupuesto");
    }
  }

  async function handleDeleteLine(accountCode: string) {
    if (!selected) return;
    if (!(await confirmDialog(`¿Quitar la cuenta ${accountCode} de este presupuesto?`, { tone: "danger" }))) return;
    try {
      await deleteBudgetLine(selected, accountCode);
      toast.success("Línea eliminada.");
      refreshLines();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar la línea");
    }
  }

  async function handleApprove() {
    if (!selected) return;
    try {
      await approveBudget(selected);
      toast.success("Presupuesto aprobado.");
      refreshBudgets();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo aprobar");
    }
  }

  async function handleCompare() {
    if (!selected) return;
    setShowCompare(true);
    setCompare(null);
    try {
      setCompare(await compareBudget(selected));
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo generar el comparativo");
    }
  }

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Presupuestos" }]} />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Calculator className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Presupuestos
        </h1>
        <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowNew(true)}>Nuevo presupuesto</Button>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {showNew && (
        <Card className="mb-3 p-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <Input label="Año" type="number" value={newYear} onChange={(e) => setNewYear(Number(e.target.value))} />
            <div className="sm:col-span-2">
              <Input label="Nombre" required value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="Ej. Presupuesto 2026" />
            </div>
          </div>
          <div className="mt-3 flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={() => setShowNew(false)}>Cancelar</Button>
            <Button type="button" disabled={!newName} loading={saving} onClick={handleCreateBudget}>Crear</Button>
          </div>
        </Card>
      )}

      {budgets === null ? (
        <div className="flex min-h-32 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
      ) : budgets.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Todavía no tienes presupuestos.</p>
      ) : (
        <>
          <div className="mb-3 flex flex-wrap items-center gap-2">
            {budgets.map((b) => (
              <button
                key={b.id}
                type="button"
                onClick={() => setSelected(b.id)}
                className={`rounded border px-3 py-1.5 text-xs transition-colors ${selected === b.id ? "border-(--accent-primary) bg-(--bg-selected) text-(--accent-primary)" : "border-(--border-color) text-(--text-secondary) hover:bg-(--bg-hover)"}`}
              >
                {b.name} <span className="text-(--text-muted)">({b.year})</span>
              </button>
            ))}
          </div>

          {selectedBudget && (
            <>
              <div className="mb-3 flex items-center justify-between">
                <StatusPill tone={selectedBudget.status === "APPROVED" ? "success" : selectedBudget.status === "CLOSED" ? "neutral" : "warning"} label={selectedBudget.status === "DRAFT" ? "Borrador" : selectedBudget.status === "APPROVED" ? "Aprobado" : "Cerrado"} />
                <div className="flex gap-2">
                  {selectedBudget.status === "DRAFT" && (
                    <>
                      <button type="button" title="Renombrar" aria-label={`Renombrar ${selectedBudget.name}`}
                        onClick={() => { setRenameValue(selectedBudget.name); setRenaming(true); }}
                        className="rounded p-1.5 text-(--text-muted) transition-colors hover:bg-(--bg-hover) hover:text-(--text-primary)">
                        <Pencil className="h-3.5 w-3.5" />
                      </button>
                      <button type="button" title="Eliminar presupuesto" aria-label={`Eliminar ${selectedBudget.name}`}
                        onClick={handleDeleteBudget}
                        className="rounded p-1.5 text-(--color-danger) transition-colors hover:bg-(--bg-hover)">
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </>
                  )}
                  <Button type="button" variant="secondary" onClick={handleCompare}>Comparar vs. real</Button>
                  {selectedBudget.status === "DRAFT" && <Button type="button" variant="success" onClick={handleApprove}>Aprobar</Button>}
                </div>
              </div>

              {renaming && (
                <Card className="mb-3 p-4">
                  <div className="flex items-end gap-2">
                    <div className="flex-1"><Input label="Nombre del presupuesto" value={renameValue} onChange={(e) => setRenameValue(e.target.value)} /></div>
                    <Button type="button" variant="secondary" onClick={() => setRenaming(false)}>Cancelar</Button>
                    <Button type="button" disabled={!renameValue} loading={savingRename} onClick={handleRename}>Guardar</Button>
                  </div>
                </Card>
              )}

              {selectedBudget.status === "DRAFT" && (
                <Card className="mb-3 p-4">
                  <div className="mb-2 w-72"><Combobox label="Cuenta" value={lineAccount} onChange={setLineAccount} options={accountOptions} placeholder="Buscar cuenta…" /></div>
                  <div className="grid grid-cols-4 gap-2 sm:grid-cols-6 lg:grid-cols-12">
                    {MONTHS.map((m, i) => (
                      <Input key={m} label={m} type="number" min="0" value={lineMonths[i]} onChange={(e) => setLineMonths((prev) => prev.map((v, idx) => (idx === i ? e.target.value : v)))} />
                    ))}
                  </div>
                  <div className="mt-3 flex justify-end">
                    <Button type="button" variant="secondary" disabled={!lineAccount} loading={savingLine} onClick={handleSetLine}>Guardar línea</Button>
                  </div>
                </Card>
              )}

              {!showCompare && lines && (
                lines.length === 0 ? (
                  <p className="text-xs text-(--text-secondary)">Sin líneas presupuestadas todavía.</p>
                ) : (
                  <div className="overflow-x-auto rounded border border-(--border-color)">
                    <table className="w-full text-left text-xs">
                      <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                        <tr>
                          <th className="px-3 py-2 font-medium">Cuenta</th>
                          {MONTHS.map((m) => <th key={m} className="px-2 py-2 text-right font-medium">{m}</th>)}
                          <th className="px-3 py-2 text-right font-medium">Total</th>
                          {selectedBudget.status === "DRAFT" && <th className="px-3 py-2" />}
                        </tr>
                      </thead>
                      <tbody>
                        {lines.map((l, i) => (
                          <tr key={l.account_code} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                            <td className="px-3 py-2 text-(--text-primary)">{l.account_code} — {l.account_name}</td>
                            {l.months.map((m, idx) => <td key={idx} className="px-2 py-2 text-right font-mono text-(--text-secondary)">{m ? money(m) : "—"}</td>)}
                            <td className="px-3 py-2 text-right font-mono font-semibold text-(--text-primary)">{money(l.total)}</td>
                            {selectedBudget.status === "DRAFT" && (
                              <td className="px-3 py-2 text-right">
                                <button type="button" title="Quitar cuenta" aria-label={`Quitar ${l.account_code}`} onClick={() => handleDeleteLine(l.account_code)}
                                  className="rounded p-1 text-(--color-danger) transition-colors hover:bg-(--bg-hover)">
                                  <Trash2 className="h-3.5 w-3.5" />
                                </button>
                              </td>
                            )}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )
              )}

              {showCompare && (
                compare === null ? (
                  <div className="flex min-h-24 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
                ) : compare.length === 0 ? (
                  <p className="text-xs text-(--text-secondary)">Sin líneas para comparar.</p>
                ) : (
                  <div className="overflow-x-auto rounded border border-(--border-color)">
                    <table className="w-full text-left text-xs">
                      <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                        <tr>
                          <th className="px-3 py-2 font-medium">Cuenta</th>
                          <th className="px-3 py-2 text-right font-medium">Presupuestado</th>
                          <th className="px-3 py-2 text-right font-medium">Real</th>
                          <th className="px-3 py-2 text-right font-medium">Diferencia</th>
                        </tr>
                      </thead>
                      <tbody>
                        {compare.map((r, i) => {
                          const budgeted = r.budgeted_months.reduce((s, m) => s + m, 0);
                          const actual = r.actual_months.reduce((s, m) => s + m, 0);
                          return (
                            <tr key={r.account_code} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                              <td className="px-3 py-2 text-(--text-primary)">{r.account_code} — {r.account_name}</td>
                              <td className="px-3 py-2 text-right font-mono text-(--text-secondary)">{money(budgeted)}</td>
                              <td className="px-3 py-2 text-right font-mono text-(--text-secondary)">{money(actual)}</td>
                              <td className={`px-3 py-2 text-right font-mono ${actual > budgeted ? "text-(--color-danger-text)" : "text-(--color-success-text)"}`}>{money(actual - budgeted)}</td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )
              )}
            </>
          )}
        </>
      )}
    </div>
  );
}
