import { useEffect, useMemo, useState } from "react";
import { BookText, Plus, Trash2 } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import { getJournal, listAccounts, postJournal, voidJournal } from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { amountToCents, formatCOP } from "../lib/currency";
import { formatDateOnly } from "../lib/dateFormat";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { Account, JournalEntry, JournalStatus } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Combobox } from "../components/ui/Combobox";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

const STATUS_LABEL: Record<JournalStatus, string> = { DRAFT: "Borrador", POSTED: "Contabilizado", VOID: "Anulado" };
const STATUS_TONE: Record<JournalStatus, StatusTone> = { DRAFT: "neutral", POSTED: "success", VOID: "danger" };

interface LineForm {
  account_code: string;
  debit: string;  // pesos, como string de <Input>
  credit: string; // pesos
  description: string;
}

const EMPTY_LINE: LineForm = { account_code: "", debit: "", credit: "", description: "" };

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

export function AccountingJournalEditorPage() {
  const { id } = useParams<{ id: string }>();
  const isNew = id === "new";
  const navigate = useNavigate();
  const confirmDialog = useConfirm();
  const toast = useToast();

  const [accounts, setAccounts] = useState<Account[]>([]);
  const [entry, setEntry] = useState<JournalEntry | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);

  const [date, setDate] = useState(todayISO());
  const [description, setDescription] = useState("");
  const [lines, setLines] = useState<LineForm[]>([{ ...EMPTY_LINE }, { ...EMPTY_LINE }]);

  useEffect(() => {
    listAccounts().then(setAccounts).catch(() => setAccounts([]));
  }, []);

  useEffect(() => {
    if (isNew || !id) return;
    setLoading(true);
    getJournal(id)
      .then(setEntry)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el asiento"))
      .finally(() => setLoading(false));
  }, [id, isNew]);

  const accountOptions = useMemo(
    () => accounts.filter((a) => a.is_posting && a.is_active).map((a) => ({ value: a.code, label: `${a.code} — ${a.name}` })),
    [accounts],
  );

  const accountName = useMemo(() => {
    const map = new Map(accounts.map((a) => [a.code, a.name]));
    return (code: string) => map.get(code) ?? "";
  }, [accounts]);

  function updateLine(i: number, patch: Partial<LineForm>) {
    setLines((prev) => prev.map((l, idx) => (idx === i ? { ...l, ...patch } : l)));
  }

  function addLine() {
    setLines((prev) => [...prev, { ...EMPTY_LINE }]);
  }

  function removeLine(i: number) {
    setLines((prev) => prev.filter((_, idx) => idx !== i));
  }

  const totals = lines.reduce(
    (acc, l) => ({ debit: acc.debit + amountToCents(l.debit), credit: acc.credit + amountToCents(l.credit) }),
    { debit: 0, credit: 0 },
  );
  const balanced = totals.debit === totals.credit && totals.debit > 0;
  const canSave =
    description.trim() !== "" &&
    lines.length >= 2 &&
    lines.every((l) => l.account_code && ((amountToCents(l.debit) > 0) !== (amountToCents(l.credit) > 0))) &&
    balanced;

  async function handleSave() {
    setError(null);
    setSaving(true);
    try {
      const created = await postJournal({
        date,
        description,
        lines: lines.map((l) => ({
          account_code: l.account_code,
          debit_cents: amountToCents(l.debit),
          credit_cents: amountToCents(l.credit),
          description: l.description || undefined,
        })),
      });
      toast.success("Asiento contabilizado.");
      navigate(`/accounting/journals/${created.id}`, { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo registrar el asiento");
    } finally {
      setSaving(false);
    }
  }

  async function handleVoid() {
    if (!entry) return;
    if (!(await confirmDialog(`¿Anular el asiento "${entry.description}"? Esta acción no se puede deshacer.`, { tone: "danger" }))) return;
    try {
      await voidJournal(entry.id);
      toast.success("Asiento anulado.");
      const updated = await getJournal(entry.id);
      setEntry(updated);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo anular el asiento");
    }
  }

  if (!isNew && loading) {
    return (
      <div className="p-4">
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      </div>
    );
  }

  if (!isNew && !entry) {
    return (
      <div className="p-4">
        <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Asientos", to: "/accounting/journals" }, { label: "No encontrado" }]} />
        {error && <Banner tone="danger">{error}</Banner>}
      </div>
    );
  }

  // ── Vista de detalle (asiento existente) ──────────────────────────────────────
  if (entry) {
    const entryTotals = entry.lines.reduce((acc, l) => ({ debit: acc.debit + l.debit, credit: acc.credit + l.credit }), { debit: 0, credit: 0 });
    return (
      <div className="p-4">
        <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Asientos", to: "/accounting/journals" }, { label: entry.voucher_number || entry.description, muted: true }]} />
        <div className="mb-3 flex items-center justify-between">
          <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
            <BookText className="h-4 w-4 shrink-0 text-(--accent-primary)" />
            {entry.description}
            <StatusPill tone={STATUS_TONE[entry.status]} label={STATUS_LABEL[entry.status]} />
          </h1>
          {entry.status === "POSTED" && (
            <Button type="button" variant="danger" onClick={handleVoid}>Anular asiento</Button>
          )}
        </div>

        {error && <Banner tone="danger">{error}</Banner>}

        <Card className="mb-3 grid grid-cols-2 gap-3 p-4 text-xs sm:grid-cols-4">
          <div><span className="text-(--text-secondary)">Fecha</span><p className="text-(--text-primary)">{formatDateOnly(entry.date)}</p></div>
          <div><span className="text-(--text-secondary)">Comprobante</span><p className="font-mono text-(--text-primary)">{entry.voucher_number || "—"}</p></div>
          <div><span className="text-(--text-secondary)">Origen</span><p className="text-(--text-primary)">{entry.source}</p></div>
          <div><span className="text-(--text-secondary)">Tipo</span><p className="text-(--text-primary)">{entry.entry_type}</p></div>
        </Card>

        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Cuenta</th>
                <th className="px-3 py-2 font-medium">Descripción</th>
                <th className="px-3 py-2 font-medium">Débito</th>
                <th className="px-3 py-2 font-medium">Crédito</th>
              </tr>
            </thead>
            <tbody>
              {entry.lines.map((l, i) => (
                <tr key={l.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 font-mono text-(--text-primary)">{l.account_code} <span className="font-sans text-(--text-secondary)">— {accountName(l.account_code)}</span></td>
                  <td className="px-3 py-2 text-(--text-secondary)">{l.description || "—"}</td>
                  <td className="px-3 py-2 font-mono text-(--text-primary)">{l.debit ? formatCOP.format(l.debit / 100) : "—"}</td>
                  <td className="px-3 py-2 font-mono text-(--text-primary)">{l.credit ? formatCOP.format(l.credit / 100) : "—"}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="border-t border-(--border-color) font-semibold">
                <td className="px-3 py-2" colSpan={2}>Total</td>
                <td className="px-3 py-2 font-mono text-(--text-primary)">{formatCOP.format(entryTotals.debit / 100)}</td>
                <td className="px-3 py-2 font-mono text-(--text-primary)">{formatCOP.format(entryTotals.credit / 100)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    );
  }

  // ── Formulario de asiento manual nuevo ────────────────────────────────────────
  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Asientos", to: "/accounting/journals" }, { label: "Nuevo asiento" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <BookText className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Nuevo asiento manual
      </h1>

      {error && <Banner tone="danger">{error}</Banner>}

      <Card className="p-4">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <Input type="date" label="Fecha" required value={date} onChange={(e) => setDate(e.target.value)} />
          <div className="sm:col-span-2">
            <Input label="Descripción" required value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Ej. Causación de arriendo de enero" />
          </div>
        </div>

        <div className="mt-4 overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-2 py-2 font-medium">Cuenta</th>
                <th className="px-2 py-2 font-medium">Descripción</th>
                <th className="w-32 px-2 py-2 font-medium">Débito</th>
                <th className="w-32 px-2 py-2 font-medium">Crédito</th>
                <th className="w-8 px-2 py-2" />
              </tr>
            </thead>
            <tbody>
              {lines.map((l, i) => (
                <tr key={i} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-2 py-1.5">
                    <Combobox value={l.account_code} onChange={(v) => updateLine(i, { account_code: v })} options={accountOptions} placeholder="Buscar cuenta..." />
                  </td>
                  <td className="px-2 py-1.5">
                    <input
                      value={l.description}
                      onChange={(e) => updateLine(i, { description: e.target.value })}
                      className="w-full rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-xs text-(--text-primary)"
                    />
                  </td>
                  <td className="px-2 py-1.5">
                    <input
                      type="number" min="0" step="0.01"
                      value={l.debit}
                      onChange={(e) => updateLine(i, { debit: e.target.value, credit: e.target.value ? "" : l.credit })}
                      className="w-full rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-right font-mono text-xs text-(--text-primary)"
                    />
                  </td>
                  <td className="px-2 py-1.5">
                    <input
                      type="number" min="0" step="0.01"
                      value={l.credit}
                      onChange={(e) => updateLine(i, { credit: e.target.value, debit: e.target.value ? "" : l.debit })}
                      className="w-full rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-right font-mono text-xs text-(--text-primary)"
                    />
                  </td>
                  <td className="px-2 py-1.5">
                    {lines.length > 2 && (
                      <button type="button" onClick={() => removeLine(i)} className="rounded p-1 text-(--color-danger) hover:bg-(--bg-hover)">
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="border-t border-(--border-color) font-semibold">
                <td className="px-2 py-2" colSpan={2}>
                  <button type="button" onClick={addLine} className="inline-flex items-center gap-1 text-(--accent-primary) hover:underline">
                    <Plus className="h-3 w-3" /> Agregar línea
                  </button>
                </td>
                <td className="px-2 py-2 text-right font-mono text-(--text-primary)">{formatCOP.format(totals.debit / 100)}</td>
                <td className="px-2 py-2 text-right font-mono text-(--text-primary)">{formatCOP.format(totals.credit / 100)}</td>
                <td />
              </tr>
            </tfoot>
          </table>
        </div>

        {!balanced && totals.debit + totals.credit > 0 && (
          <p className="mt-2 text-xs text-(--color-danger-text)">El asiento debe estar balanceado: la suma de débitos debe ser igual a la de créditos.</p>
        )}

        <div className="mt-4 flex justify-end gap-2 border-t border-(--border-light) pt-3">
          <Button type="button" variant="secondary" onClick={() => navigate("/accounting/journals")}>Cancelar</Button>
          <Button type="button" disabled={!canSave} loading={saving} onClick={handleSave}>Contabilizar</Button>
        </div>
      </Card>
    </div>
  );
}
