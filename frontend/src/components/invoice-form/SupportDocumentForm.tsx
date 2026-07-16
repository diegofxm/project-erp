import { useEffect, useState } from "react";
import { Trash2 } from "lucide-react";
import { listCurrencies } from "../../lib/catalogs";
import { lineToInput } from "../../lib/documents";
import { listNumberingRanges } from "../../lib/numberingRanges";
import { useCatalog } from "../../lib/useCatalog";
import type {
  Document,
  DocumentLineInput,
  IssueSupportDocumentPayload,
  NumberingRange,
  PaymentMean,
  Tax,
  VendorPayload,
} from "../../lib/types";
import { Button } from "../ui/Button";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { VendorSection } from "./VendorSection";
import { LineItemsEditor } from "./LineItemsEditor";
import { PaymentMeansEditor } from "./PaymentMeansEditor";
import { TotalsSummary } from "./TotalsSummary";

const SUPPORT_DOCUMENT_DIAN_TYPE = "05";

const OPERATION_TYPE_OPTIONS = [
  { code: "10", label: "Residente (10)" },
  { code: "11", label: "No Residente (11)" },
];

const WITHHOLDING_TYPE_OPTIONS = [
  { code: "05", name: "ReteIVA", label: "ReteIVA (05)" },
  { code: "06", name: "ReteRenta", label: "ReteRenta (06)" },
];

const NEW_VENDOR: VendorPayload = {
  identification: { number: "", type_code: "13" },
  name: "",
  entity_type_code: "1",
  tax_scheme_code: "ZZ",
  liability_codes: ["R-99-PN"],
  tax_regime_code: "49",
};

const EMPTY_WITHHOLDING: Tax = {
  taxable_amount_cents: 0,
  tax_amount_cents: 0,
  percent: 0,
  type_code: "06",
  type_name: "ReteRenta",
};

interface SupportDocumentFormProps {
  initial: Document | null;
  onSubmit: (payload: IssueSupportDocumentPayload) => void;
  onCancel: () => void;
  loading: boolean;
}

export function SupportDocumentForm({ initial, onSubmit, onCancel, loading }: SupportDocumentFormProps) {
  const [ranges, setRanges] = useState<NumberingRange[]>([]);
  const [loadingRanges, setLoadingRanges] = useState(true);
  const { data: currencies, loading: loadingCurrencies } = useCatalog(listCurrencies);

  const [numberingRangeId, setNumberingRangeId] = useState(initial?.numbering_range_id ?? "");
  const [operationTypeCode, setOperationTypeCode] = useState(initial?.operation_type_code ?? "10");
  const [vendor, setVendor] = useState<VendorPayload>(initial?.vendor ?? NEW_VENDOR);
  const [vendorId, setVendorId] = useState(initial?.vendor_id ?? "");
  const [lines, setLines] = useState<DocumentLineInput[]>(initial?.lines.map(lineToInput) ?? []);
  const [paymentMeans, setPaymentMeans] = useState<PaymentMean[]>(initial?.payment_means ?? []);
  const [withholdingTaxes, setWithholdingTaxes] = useState<Tax[]>(initial?.withholding_taxes ?? []);
  const [note, setNote] = useState(initial?.note ?? "");
  const [currencyCode, setCurrencyCode] = useState(initial?.currency_code ?? "COP");

  useEffect(() => {
    listNumberingRanges(SUPPORT_DOCUMENT_DIAN_TYPE)
      .then(setRanges)
      .catch(() => setRanges([]))
      .finally(() => setLoadingRanges(false));
  }, []);

  function addWithholding() {
    setWithholdingTaxes((prev) => [...prev, { ...EMPTY_WITHHOLDING }]);
  }

  function removeWithholding(index: number) {
    setWithholdingTaxes((prev) => prev.filter((_, i) => i !== index));
  }

  function updateWithholding(index: number, field: keyof Tax, raw: string) {
    setWithholdingTaxes((prev) =>
      prev.map((t, i) => {
        if (i !== index) return t;
        if (field === "type_code") {
          const opt = WITHHOLDING_TYPE_OPTIONS.find((o) => o.code === raw);
          return { ...t, type_code: raw, type_name: opt?.name ?? raw };
        }
        const num = parseInt(raw, 10);
        if (field === "taxable_amount_cents" || field === "tax_amount_cents") {
          return { ...t, [field]: isNaN(num) ? 0 : num };
        }
        if (field === "percent") {
          const f = parseFloat(raw);
          return { ...t, percent: isNaN(f) ? 0 : f };
        }
        return t;
      })
    );
  }

  function handleVendorChange(next: VendorPayload, nextVendorId: string) {
    setVendor(next);
    setVendorId(nextVendorId);
  }

  function handleSubmit() {
    onSubmit({
      numbering_range_id: numberingRangeId,
      vendor_id: vendorId || undefined,
      vendor,
      lines,
      payment_means: paymentMeans.length > 0 ? paymentMeans : undefined,
      note: note || undefined,
      currency_code: currencyCode,
      operation_type_code: operationTypeCode,
      withholding_taxes: withholdingTaxes.length > 0 ? withholdingTaxes : undefined,
    });
  }

  const selectedRange = ranges.find((r) => r.id === numberingRangeId);
  const selectableRanges = ranges.filter((r) => r.status === "active" || r.id === numberingRangeId);
  const canSubmit =
    numberingRangeId !== "" &&
    vendor.identification.number.trim() !== "" &&
    lines.length > 0 &&
    paymentMeans.length > 0;

  return (
    <div className="flex flex-col gap-4 p-4">
      {/* Cabecera: rango, tipo operación, moneda, nota */}
      <div className="grid grid-cols-12 gap-3">
        <div className="col-span-6">
          <Select
            label="Rango de numeración"
            required
            value={numberingRangeId}
            onChange={(e) => setNumberingRangeId(e.target.value)}
            disabled={loadingRanges}
          >
            {loadingRanges ? (
              <option>Cargando…</option>
            ) : (
              <>
                <option value="">Selecciona…</option>
                {selectableRanges.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.prefix || "(sin prefijo)"}
                  </option>
                ))}
              </>
            )}
          </Select>
          {selectedRange && (
            <p className="mt-1 text-xs text-(--text-muted)">
              Próximo número: {selectedRange.prefix}{selectedRange.current_number + 1}
            </p>
          )}
          {!loadingRanges && selectableRanges.length === 0 && (
            <p className="mt-1 text-xs text-(--text-muted)">
              No hay un rango activo para Documento Soporte — créalo en Configuración → Empresa.
            </p>
          )}
        </div>
        <div className="col-span-3">
          <Select
            label="Tipo de operación"
            value={operationTypeCode}
            onChange={(e) => setOperationTypeCode(e.target.value)}
          >
            {OPERATION_TYPE_OPTIONS.map((o) => (
              <option key={o.code} value={o.code}>{o.label}</option>
            ))}
          </Select>
        </div>
        <div className="col-span-3">
          <Select
            label="Moneda"
            disabled={loadingCurrencies}
            value={currencyCode}
            onChange={(e) => setCurrencyCode(e.target.value)}
          >
            {loadingCurrencies ? (
              <option>Cargando…</option>
            ) : (
              currencies.map((c) => (
                <option key={c.code} value={c.code}>
                  {c.code} — {c.name}
                </option>
              ))
            )}
          </Select>
        </div>
        <div className="col-span-12">
          <Input label="Nota (opcional)" value={note} onChange={(e) => setNote(e.target.value)} />
        </div>
      </div>

      {/* Tercero no obligado */}
      <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
        <h2 className="text-xs font-semibold text-(--text-primary)">Tercero no obligado a facturar</h2>
        <VendorSection value={vendor} vendorId={vendorId} onChange={handleVendorChange} />
      </section>

      {/* Líneas */}
      <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
        <h2 className="text-xs font-semibold text-(--text-primary)">Líneas</h2>
        <LineItemsEditor lines={lines} onChange={setLines} />
      </section>

      {/* Forma de pago */}
      <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
        <h2 className="text-xs font-semibold text-(--text-primary)">Forma de pago</h2>
        <PaymentMeansEditor paymentMeans={paymentMeans} onChange={setPaymentMeans} />
      </section>

      {/* Retenciones */}
      <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
        <div className="flex items-center justify-between">
          <h2 className="text-xs font-semibold text-(--text-primary)">Retenciones</h2>
          <Button type="button" variant="secondary" onClick={addWithholding}>
            + Agregar retención
          </Button>
        </div>
        {withholdingTaxes.length === 0 ? (
          <p className="text-xs text-(--text-muted)">Sin retenciones — opcional.</p>
        ) : (
          <div className="overflow-x-auto rounded border border-(--border-color)">
            <table className="w-full text-xs">
              <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                <tr>
                  <th className="px-2 py-1.5 text-left font-medium">Tipo</th>
                  <th className="px-2 py-1.5 text-left font-medium">Base (centavos)</th>
                  <th className="px-2 py-1.5 text-left font-medium">Retención (centavos)</th>
                  <th className="px-2 py-1.5 text-left font-medium">%</th>
                  <th className="px-2 py-1.5" />
                </tr>
              </thead>
              <tbody>
                {withholdingTaxes.map((t, i) => (
                  <tr key={i} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                    <td className="px-2 py-1">
                      <Select value={t.type_code} onChange={(e) => updateWithholding(i, "type_code", e.target.value)}>
                        {WITHHOLDING_TYPE_OPTIONS.map((o) => (
                          <option key={o.code} value={o.code}>{o.label}</option>
                        ))}
                      </Select>
                    </td>
                    <td className="px-2 py-1">
                      <Input
                        type="number"
                        min="0"
                        value={t.taxable_amount_cents}
                        onChange={(e) => updateWithholding(i, "taxable_amount_cents", e.target.value)}
                      />
                    </td>
                    <td className="px-2 py-1">
                      <Input
                        type="number"
                        min="0"
                        value={t.tax_amount_cents}
                        onChange={(e) => updateWithholding(i, "tax_amount_cents", e.target.value)}
                      />
                    </td>
                    <td className="px-2 py-1">
                      <Input
                        type="number"
                        min="0"
                        step="0.01"
                        value={t.percent}
                        onChange={(e) => updateWithholding(i, "percent", e.target.value)}
                      />
                    </td>
                    <td className="px-2 py-1">
                      <button
                        type="button"
                        onClick={() => removeWithholding(i)}
                        className="rounded p-0.5 text-(--text-muted) hover:text-(--danger) transition-colors"
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
      </section>

      {/* Totales alineados a la derecha */}
      <section className="grid grid-cols-12 gap-3 border-t border-(--border-color) pt-3">
        <div className="col-span-4 col-start-9">
          <TotalsSummary lines={lines} withholdingTaxes={withholdingTaxes} />
        </div>
      </section>

      <div className="flex gap-2">
        <Button type="button" variant="secondary" onClick={onCancel} className="flex-1">
          Cancelar
        </Button>
        <Button type="button" disabled={!canSubmit} loading={loading} onClick={handleSubmit} className="flex-1">
          {initial ? "Guardar borrador" : "Crear borrador"}
        </Button>
      </div>
    </div>
  );
}
