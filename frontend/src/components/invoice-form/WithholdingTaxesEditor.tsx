import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { amountToCents, formatCOP } from "../../lib/currency";
import type { Tax } from "../../lib/types";
import { Button } from "../ui/Button";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";

const WITHHOLDING_TYPE_OPTIONS = [
  { code: "05", name: "ReteIVA" },
  { code: "06", name: "ReteRenta" },
];

interface Draft {
  typeCode: string;
  taxableAmount: string;
  taxAmount: string;
  percent: string;
}

const EMPTY_DRAFT: Draft = { typeCode: "06", taxableAmount: "", taxAmount: "", percent: "" };

interface WithholdingTaxesEditorProps {
  taxes: Tax[];
  onChange: (taxes: Tax[]) => void;
}

export function WithholdingTaxesEditor({ taxes, onChange }: WithholdingTaxesEditorProps) {
  const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
  const [showForm, setShowForm] = useState(false);

  function handleAdd() {
    const typeOpt = WITHHOLDING_TYPE_OPTIONS.find((o) => o.code === draft.typeCode);
    onChange([
      ...taxes,
      {
        type_code: draft.typeCode,
        type_name: typeOpt?.name ?? draft.typeCode,
        taxable_amount_cents: amountToCents(draft.taxableAmount),
        tax_amount_cents: amountToCents(draft.taxAmount),
        percent: Number(draft.percent || 0),
      },
    ]);
    setDraft(EMPTY_DRAFT);
    setShowForm(false);
  }

  function handleCancel() {
    setDraft(EMPTY_DRAFT);
    setShowForm(false);
  }

  function handleRemove(index: number) {
    onChange(taxes.filter((_, i) => i !== index));
  }

  const canAdd = amountToCents(draft.taxableAmount) > 0 && amountToCents(draft.taxAmount) > 0;

  return (
    <div className="flex flex-col gap-3">
      {/* Tabla de retenciones agregadas */}
      {taxes.length > 0 && (
        <div className="overflow-hidden rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Tipo</th>
                <th className="px-3 py-2 font-medium">Base imponible</th>
                <th className="px-3 py-2 font-medium">Valor retención</th>
                <th className="px-3 py-2 font-medium">%</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {taxes.map((t, i) => (
                <tr key={i} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 font-medium text-(--text-primary)">
                    {WITHHOLDING_TYPE_OPTIONS.find((o) => o.code === t.type_code)?.name ?? t.type_code}
                  </td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">
                    {formatCOP.format(t.taxable_amount_cents / 100)}
                  </td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">
                    {formatCOP.format(t.tax_amount_cents / 100)}
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">{t.percent}%</td>
                  <td className="px-3 py-2">
                    <button
                      type="button"
                      title="Quitar retención"
                      onClick={() => handleRemove(i)}
                      className="rounded p-1 text-(--color-danger) hover:bg-(--bg-hover)"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Estado vacío */}
      {taxes.length === 0 && !showForm && (
        <p className="text-xs text-(--text-muted)">Sin retenciones — campo opcional.</p>
      )}

      {/* Botón o formulario */}
      {!showForm ? (
        <Button
          type="button"
          variant="secondary"
          icon={<Plus className="h-3.5 w-3.5" />}
          onClick={() => setShowForm(true)}
        >
          Agregar retención
        </Button>
      ) : (
        <div className="rounded border border-(--border-color) bg-(--bg-secondary) p-3">
          <div className="grid grid-cols-12 gap-3">
            <div className="col-span-3">
              <Select
                label="Tipo de retención"
                value={draft.typeCode}
                onChange={(e) => setDraft({ ...draft, typeCode: e.target.value })}
              >
                {WITHHOLDING_TYPE_OPTIONS.map((o) => (
                  <option key={o.code} value={o.code}>
                    {o.name}
                  </option>
                ))}
              </Select>
            </div>
            <div className="col-span-3">
              <Input
                label="Base imponible (COP)"
                type="number"
                min="0"
                step="0.01"
                value={draft.taxableAmount}
                onChange={(e) => setDraft({ ...draft, taxableAmount: e.target.value })}
              />
            </div>
            <div className="col-span-3">
              <Input
                label="Valor retención (COP)"
                type="number"
                min="0"
                step="0.01"
                value={draft.taxAmount}
                onChange={(e) => setDraft({ ...draft, taxAmount: e.target.value })}
              />
            </div>
            <div className="col-span-3">
              <Input
                label="Porcentaje (%)"
                type="number"
                min="0"
                step="0.01"
                value={draft.percent}
                onChange={(e) => setDraft({ ...draft, percent: e.target.value })}
              />
            </div>
            <div className="col-span-12 flex items-center justify-end gap-2 pt-1">
              <Button type="button" variant="ghost" onClick={handleCancel}>
                Cancelar
              </Button>
              <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} disabled={!canAdd} onClick={handleAdd}>
                Agregar
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
