import { useEffect, useMemo, useState } from "react";
import { Landmark, Plus, Link2, Pencil, Archive } from "lucide-react";
import {
  addStatementLine, createBankAccount, deactivateBankAccount, getCandidates, listAccounts, listBankAccounts,
  listStatement, reconcile, updateBankAccount,
} from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { formatDateOnly, todayColombiaISO } from "../lib/dateFormat";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { Account, BankAccount, ReconciliationCandidate, StatementLine } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Combobox } from "../components/ui/Combobox";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill } from "../components/ui/StatusPill";

function money(v: number): string {
  return formatCOP.format(v / 100);
}

function todayISO(): string {
  return todayColombiaISO();
}

export function AccountingBankPage() {
  const toast = useToast();
  const confirmDialog = useConfirm();
  const [accounts, setAccounts] = useState<BankAccount[] | null>(null);
  const [pucAccounts, setPucAccounts] = useState<Account[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [statement, setStatement] = useState<StatementLine[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [showNewAccount, setShowNewAccount] = useState(false);
  const [newName, setNewName] = useState("");
  const [newBankName, setNewBankName] = useState("");
  const [newAccountNo, setNewAccountNo] = useState("");
  const [newAccountCode, setNewAccountCode] = useState("111005");
  const [savingAccount, setSavingAccount] = useState(false);

  const [showNewLine, setShowNewLine] = useState(false);
  const [lineDate, setLineDate] = useState(todayISO());
  const [lineDesc, setLineDesc] = useState("");
  const [lineDebit, setLineDebit] = useState("");
  const [lineCredit, setLineCredit] = useState("");
  const [lineRef, setLineRef] = useState("");
  const [savingLine, setSavingLine] = useState(false);

  const [candidateFor, setCandidateFor] = useState<StatementLine | null>(null);
  const [candidates, setCandidates] = useState<ReconciliationCandidate[] | null>(null);

  const [editingAccount, setEditingAccount] = useState<BankAccount | null>(null);
  const [editName, setEditName] = useState("");
  const [editBankName, setEditBankName] = useState("");
  const [editAccountNo, setEditAccountNo] = useState("");
  const [savingEdit, setSavingEdit] = useState(false);

  function refreshAccounts() {
    listBankAccounts()
      .then((list) => { setAccounts(list); if (!selected && list.length > 0) setSelected(list[0].id); })
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar las cuentas bancarias"));
  }

  useEffect(() => {
    refreshAccounts();
    listAccounts().then(setPucAccounts).catch(() => setPucAccounts([]));
  }, []);

  useEffect(() => {
    if (!selected) { setStatement(null); return; }
    listStatement(selected).then(setStatement).catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el extracto"));
  }, [selected]);

  const bankAccountOptions = useMemo(
    () => pucAccounts.filter((a) => a.is_posting && (a.code.startsWith("11") || a.category === "Activo")).map((a) => ({ value: a.code, label: `${a.code} — ${a.name}` })),
    [pucAccounts],
  );

  async function handleCreateAccount() {
    if (!newName || !newAccountNo || !newAccountCode) return;
    setSavingAccount(true);
    try {
      await createBankAccount({ name: newName, bank_name: newBankName, account_no: newAccountNo, account_code: newAccountCode });
      toast.success("Cuenta bancaria creada.");
      setShowNewAccount(false);
      setNewName(""); setNewBankName(""); setNewAccountNo("");
      refreshAccounts();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo crear la cuenta bancaria");
    } finally {
      setSavingAccount(false);
    }
  }

  function openEdit(a: BankAccount) {
    setEditingAccount(a);
    setEditName(a.name);
    setEditBankName(a.bank_name);
    setEditAccountNo(a.account_no);
  }

  async function handleSaveEdit() {
    if (!editingAccount || !editName || !editAccountNo) return;
    setSavingEdit(true);
    try {
      await updateBankAccount(editingAccount.id, { name: editName, bank_name: editBankName, account_no: editAccountNo });
      toast.success("Cuenta bancaria actualizada.");
      setEditingAccount(null);
      refreshAccounts();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo actualizar la cuenta bancaria");
    } finally {
      setSavingEdit(false);
    }
  }

  async function handleDeactivate(a: BankAccount) {
    if (!(await confirmDialog(`¿Desactivar la cuenta "${a.name}"? Dejará de estar disponible para nuevos movimientos.`, { tone: "danger" }))) return;
    try {
      await deactivateBankAccount(a.id);
      toast.success("Cuenta bancaria desactivada.");
      refreshAccounts();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo desactivar la cuenta bancaria");
    }
  }

  async function handleAddLine() {
    if (!selected || !lineDesc) return;
    setSavingLine(true);
    try {
      await addStatementLine(selected, {
        date: lineDate, description: lineDesc,
        debit_cents: Math.round(Number(lineDebit || 0) * 100), credit_cents: Math.round(Number(lineCredit || 0) * 100),
        reference: lineRef,
      });
      toast.success("Movimiento agregado.");
      setShowNewLine(false);
      setLineDesc(""); setLineDebit(""); setLineCredit(""); setLineRef("");
      listStatement(selected).then(setStatement);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo agregar el movimiento");
    } finally {
      setSavingLine(false);
    }
  }

  async function openCandidates(line: StatementLine) {
    setCandidateFor(line);
    setCandidates(null);
    try {
      setCandidates(await getCandidates(line.id));
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudieron buscar candidatos");
    }
  }

  async function handleReconcile(journalLineId: string) {
    if (!candidateFor || !selected) return;
    try {
      await reconcile(candidateFor.id, journalLineId);
      toast.success("Movimiento conciliado.");
      setCandidateFor(null);
      listStatement(selected).then(setStatement);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo conciliar");
    }
  }

  const selectedAccount = accounts?.find((a) => a.id === selected) ?? null;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Bancos" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <Landmark className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Bancos y conciliación
      </h1>

      {error && <Banner tone="danger">{error}</Banner>}

      {accounts === null ? (
        <div className="flex min-h-32 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
      ) : (
        <>
          <div className="mb-3 flex flex-wrap items-center gap-2">
            {accounts.map((a) => (
              <div
                key={a.id}
                className={`flex items-center gap-1 rounded border pl-3 pr-1 py-1 text-xs transition-colors ${selected === a.id ? "border-(--accent-primary) bg-(--bg-selected) text-(--accent-primary)" : "border-(--border-color) text-(--text-secondary)"} ${!a.is_active ? "opacity-50" : ""}`}
              >
                <button type="button" onClick={() => setSelected(a.id)} className="hover:underline">
                  {a.name} <span className="font-mono text-(--text-muted)">{a.account_no}</span>
                  {!a.is_active && <span className="ml-1 text-(--text-muted)">(inactiva)</span>}
                </button>
                <button type="button" title="Editar" aria-label={`Editar ${a.name}`} onClick={() => openEdit(a)}
                  className="rounded p-1 text-(--text-muted) transition-colors hover:bg-(--bg-hover) hover:text-(--text-primary)">
                  <Pencil className="h-3 w-3" />
                </button>
                {a.is_active && (
                  <button type="button" title="Desactivar" aria-label={`Desactivar ${a.name}`} onClick={() => handleDeactivate(a)}
                    className="rounded p-1 text-(--color-danger) transition-colors hover:bg-(--bg-hover)">
                    <Archive className="h-3 w-3" />
                  </button>
                )}
              </div>
            ))}
            <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowNewAccount(true)}>
              Nueva cuenta
            </Button>
          </div>

          {showNewAccount && (
            <Card className="mb-3 p-4">
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-4">
                <Input label="Nombre" required value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="Ej. Cuenta corriente Bancolombia" />
                <Input label="Banco" value={newBankName} onChange={(e) => setNewBankName(e.target.value)} />
                <Input label="Número de cuenta" required value={newAccountNo} onChange={(e) => setNewAccountNo(e.target.value)} />
                <Combobox label="Cuenta PUC" value={newAccountCode} onChange={setNewAccountCode} options={bankAccountOptions} placeholder="Buscar cuenta…" />
              </div>
              <div className="mt-3 flex justify-end gap-2">
                <Button type="button" variant="secondary" onClick={() => setShowNewAccount(false)}>Cancelar</Button>
                <Button type="button" loading={savingAccount} disabled={!newName || !newAccountNo} onClick={handleCreateAccount}>Crear</Button>
              </div>
            </Card>
          )}

          {accounts.length === 0 && !showNewAccount ? (
            <p className="text-xs text-(--text-secondary)">Todavía no tienes cuentas bancarias registradas.</p>
          ) : selectedAccount && (
            <>
              <div className="mb-3 flex items-center justify-between">
                <h2 className="text-xs font-semibold text-(--text-primary)">Extracto — {selectedAccount.name}</h2>
                <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowNewLine(true)}>
                  Nuevo movimiento
                </Button>
              </div>

              {showNewLine && (
                <Card className="mb-3 p-4">
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-5">
                    <Input label="Fecha" type="date" value={lineDate} onChange={(e) => setLineDate(e.target.value)} />
                    <div className="sm:col-span-2">
                      <Input label="Descripción" required value={lineDesc} onChange={(e) => setLineDesc(e.target.value)} />
                    </div>
                    <Input label="Débito (entra)" type="number" min="0" value={lineDebit} onChange={(e) => { setLineDebit(e.target.value); if (e.target.value) setLineCredit(""); }} />
                    <Input label="Crédito (sale)" type="number" min="0" value={lineCredit} onChange={(e) => { setLineCredit(e.target.value); if (e.target.value) setLineDebit(""); }} />
                  </div>
                  <Input label="Referencia (opcional)" value={lineRef} onChange={(e) => setLineRef(e.target.value)} className="mt-3 max-w-xs" />
                  <div className="mt-3 flex justify-end gap-2">
                    <Button type="button" variant="secondary" onClick={() => setShowNewLine(false)}>Cancelar</Button>
                    <Button type="button" loading={savingLine} disabled={!lineDesc || (!lineDebit && !lineCredit)} onClick={handleAddLine}>Agregar</Button>
                  </div>
                </Card>
              )}

              {statement === null ? (
                <div className="flex min-h-24 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
              ) : statement.length === 0 ? (
                <p className="text-xs text-(--text-secondary)">Sin movimientos en el extracto todavía.</p>
              ) : (
                <div className="overflow-x-auto rounded border border-(--border-color)">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                      <tr>
                        <th className="px-3 py-2 font-medium">Fecha</th>
                        <th className="px-3 py-2 font-medium">Descripción</th>
                        <th className="px-3 py-2 font-medium">Débito</th>
                        <th className="px-3 py-2 font-medium">Crédito</th>
                        <th className="px-3 py-2 font-medium">Estado</th>
                        <th className="px-3 py-2" />
                      </tr>
                    </thead>
                    <tbody>
                      {statement.map((l, i) => (
                        <tr key={l.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                          <td className="px-3 py-2 text-(--text-secondary)">{formatDateOnly(l.date)}</td>
                          <td className="px-3 py-2 text-(--text-primary)">{l.description}</td>
                          <td className="px-3 py-2 font-mono text-(--text-primary)">{l.debit ? money(l.debit) : "—"}</td>
                          <td className="px-3 py-2 font-mono text-(--text-primary)">{l.credit ? money(l.credit) : "—"}</td>
                          <td className="px-3 py-2">
                            <StatusPill tone={l.is_reconciled ? "success" : "warning"} label={l.is_reconciled ? "Conciliado" : "Pendiente"} />
                          </td>
                          <td className="px-3 py-2">
                            {!l.is_reconciled && (
                              <button type="button" onClick={() => openCandidates(l)} className="flex items-center gap-1 text-(--accent-primary) hover:underline">
                                <Link2 className="h-3 w-3" /> Conciliar
                              </button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </>
      )}

      {editingAccount && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setEditingAccount(null)}>
          <Card className="w-full max-w-md p-4" onClick={(e) => e.stopPropagation()}>
            <h3 className="mb-3 text-xs font-semibold text-(--text-primary)">Editar cuenta bancaria</h3>
            <div className="flex flex-col gap-3">
              <Input label="Nombre" required value={editName} onChange={(e) => setEditName(e.target.value)} />
              <Input label="Banco" value={editBankName} onChange={(e) => setEditBankName(e.target.value)} />
              <Input label="Número de cuenta" required value={editAccountNo} onChange={(e) => setEditAccountNo(e.target.value)} />
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <Button type="button" variant="secondary" onClick={() => setEditingAccount(null)}>Cancelar</Button>
              <Button type="button" loading={savingEdit} disabled={!editName || !editAccountNo} onClick={handleSaveEdit}>Guardar</Button>
            </div>
          </Card>
        </div>
      )}

      {candidateFor && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setCandidateFor(null)}>
          <Card className="w-full max-w-lg p-4" onClick={(e) => e.stopPropagation()}>
            <h3 className="mb-1 text-xs font-semibold text-(--text-primary)">Conciliar: {candidateFor.description}</h3>
            <p className="mb-3 text-xs text-(--text-secondary)">
              Monto {money(candidateFor.debit || candidateFor.credit)} · {formatDateOnly(candidateFor.date)}
            </p>
            {candidates === null ? (
              <div className="flex min-h-16 items-center justify-center"><Spinner className="h-4 w-4 text-(--text-muted)" /></div>
            ) : candidates.length === 0 ? (
              <p className="text-xs text-(--text-secondary)">No se encontraron asientos con el mismo monto sin conciliar en un rango de ±15 días.</p>
            ) : (
              <div className="flex flex-col gap-1.5">
                {candidates.map((c) => (
                  <button
                    key={c.line_id}
                    type="button"
                    onClick={() => handleReconcile(c.line_id)}
                    className="flex items-center justify-between rounded border border-(--border-color) px-3 py-2 text-left text-xs hover:bg-(--bg-hover)"
                  >
                    <span>
                      <span className="text-(--text-primary)">{c.description}</span>{" "}
                      <span className="text-(--text-muted)">{c.voucher_number || "—"} · {formatDateOnly(c.date)}</span>
                    </span>
                    <span className="font-mono text-(--text-primary)">{money(c.debit || c.credit)}</span>
                  </button>
                ))}
              </div>
            )}
            <div className="mt-3 flex justify-end">
              <Button type="button" variant="secondary" onClick={() => setCandidateFor(null)}>Cerrar</Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}
