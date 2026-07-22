import { useEffect, useState } from "react";
import { listCurrencies } from "../../lib/catalogs";
import { lineToInput } from "../../lib/documents";
import { previewTotals } from "../../lib/invoiceMath";
import { listNumberingRanges } from "../../lib/numberingRanges";
import { useCatalog } from "../../lib/useCatalog";
import type {
  BillingReference,
  DiscrepancyResponse,
  Document,
  DocumentLineInput,
  IssueAdjustmentNotePayload,
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
import { WithholdingTaxesEditor } from "./WithholdingTaxesEditor";
import { TotalsSummary } from "./TotalsSummary";

const ADJUSTMENT_NOTE_DIAN_TYPE = "95";

const OPERATION_TYPE_OPTIONS = [
  { code: "10", label: "Residente (10)" },
  { code: "11", label: "No Residente (11)" },
];

// Códigos de motivo para Nota de Ajuste al DS (análogos a los de NC/ND para FE).
const DISCREPANCY_RESPONSE_OPTIONS = [
  { code: "1", label: "1 — Devolución de parte de los bienes" },
  { code: "2", label: "2 — Anulación de operación" },
  { code: "3", label: "3 — Rebaja o descuento parcial" },
  { code: "4", label: "4 — Ajuste de precio" },
  { code: "5", label: "5 — Otros" },
];

const NEW_VENDOR: VendorPayload = {
  identification: { number: "", type_code: "13" },
  name: "",
  tax_scheme_code: "ZZ",
  tax_regime_code: "49",
  liability_codes: ["O-49"],
};

interface AdjustmentNoteFormProps {
  initial: Document | null;
  prefill?: Document | null;
  billingReference: BillingReference;
  onSubmit: (payload: IssueAdjustmentNotePayload) => void;
  onCancel: () => void;
  loading: boolean;
  onTotalChange?: (cents: number) => void;
}

export function AdjustmentNoteForm({ initial, prefill, billingReference, onSubmit, onCancel, loading, onTotalChange }: AdjustmentNoteFormProps) {
  const [ranges, setRanges] = useState<NumberingRange[]>([]);
  const [loadingRanges, setLoadingRanges] = useState(true);
  const { data: currencies, loading: loadingCurrencies } = useCatalog(listCurrencies);

  const [numberingRangeId, setNumberingRangeId] = useState(initial?.numbering_range_id ?? "");
  const [operationTypeCode, setOperationTypeCode] = useState(initial?.operation_type_code ?? prefill?.operation_type_code ?? "10");
  const [vendor, setVendor] = useState<VendorPayload>(initial?.vendor ?? prefill?.vendor ?? NEW_VENDOR);
  const [vendorId, setVendorId] = useState(initial?.vendor_id ?? prefill?.vendor_id ?? "");
  const [lines, setLines] = useState<DocumentLineInput[]>(
    initial?.lines.map(lineToInput) ?? prefill?.lines.map(lineToInput) ?? []
  );
  const [paymentMeans, setPaymentMeans] = useState<PaymentMean[]>(
    initial?.payment_means ?? prefill?.payment_means ?? []
  );
  const [withholdingTaxes, setWithholdingTaxes] = useState<Tax[]>(
    initial?.withholding_taxes ?? prefill?.withholding_taxes ?? []
  );
  const [note, setNote] = useState(initial?.note ?? "");
  const [currencyCode, setCurrencyCode] = useState(initial?.currency_code ?? prefill?.currency_code ?? "COP");

  // Motivo del ajuste (opcional)
  const [hasDiscrepancy, setHasDiscrepancy] = useState(!!initial?.discrepancy_response);
  const [discrepancy, setDiscrepancy] = useState<DiscrepancyResponse>(
    initial?.discrepancy_response ?? {
      reference_id: billingReference.prefix + billingReference.number,
      response_code: "1",
      description: "",
    }
  );

  useEffect(() => {
    listNumberingRanges(ADJUSTMENT_NOTE_DIAN_TYPE)
      .then(setRanges)
      .catch(() => setRanges([]))
      .finally(() => setLoadingRanges(false));
  }, []);

  useEffect(() => {
    onTotalChange?.(previewTotals(lines).payableCents);
  }, [lines]); // eslint-disable-line react-hooks/exhaustive-deps

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
      billing_reference: billingReference,
      discrepancy_response: hasDiscrepancy ? discrepancy : undefined,
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
      {/* Documento Soporte de referencia — solo lectura, viene del DS origen */}
      <div className="rounded border border-(--border-color) bg-(--bg-secondary) p-3">
        <p className="text-xs font-medium text-(--text-secondary)">Documento Soporte de referencia</p>
        <p className="mt-1 font-mono text-sm text-(--text-primary)">
          {billingReference.prefix}{billingReference.number}
          {billingReference.issue_date && (
            <span className="ml-2 font-sans text-xs text-(--text-secondary)">
              — {new Date(billingReference.issue_date + "T00:00:00").toLocaleDateString("es-CO")}
            </span>
          )}
        </p>
        {billingReference.cufe && (
          <p className="mt-1 break-all font-mono text-xs text-(--text-muted)">{billingReference.cufe}</p>
        )}
      </div>

      {/* Cabecera: rango, tipo operación, moneda, nota */}
      <div className="grid grid-cols-12 gap-3">
        <div className="col-span-12 sm:col-span-6">
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
              No hay un rango activo para Nota de Ajuste — créalo en Configuración → Empresa.
            </p>
          )}
        </div>
        <div className="col-span-6 sm:col-span-3">
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
        <div className="col-span-6 sm:col-span-3">
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

      {/* Motivo del ajuste */}
      <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
        <div className="flex items-center gap-2">
          <h2 className="text-xs font-semibold text-(--text-primary)">Motivo del ajuste</h2>
          <label className="flex items-center gap-1 text-xs text-(--text-muted)">
            <input
              type="checkbox"
              checked={hasDiscrepancy}
              onChange={(e) => setHasDiscrepancy(e.target.checked)}
              className="h-3 w-3"
            />
            Incluir motivo
          </label>
        </div>
        {hasDiscrepancy && (
          <div className="grid grid-cols-12 gap-3">
            <div className="col-span-6 sm:col-span-3">
              <Select
                label="Código"
                value={discrepancy.response_code}
                onChange={(e) => setDiscrepancy((d) => ({ ...d, response_code: e.target.value }))}
              >
                {DISCREPANCY_RESPONSE_OPTIONS.map((o) => (
                  <option key={o.code} value={o.code}>{o.label}</option>
                ))}
              </Select>
            </div>
            <div className="col-span-6 sm:col-span-5">
              <Input
                label="Referencia"
                value={discrepancy.reference_id}
                onChange={(e) => setDiscrepancy((d) => ({ ...d, reference_id: e.target.value }))}
                placeholder="SEDS984000000"
              />
            </div>
            <div className="col-span-12">
              <Input
                label="Descripción"
                value={discrepancy.description}
                onChange={(e) => setDiscrepancy((d) => ({ ...d, description: e.target.value }))}
                placeholder="Descripción del ajuste"
              />
            </div>
          </div>
        )}
      </section>

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
        <h2 className="text-xs font-semibold text-(--text-primary)">Retenciones</h2>
        <WithholdingTaxesEditor taxes={withholdingTaxes} onChange={setWithholdingTaxes} />
      </section>

      {/* Totales */}
      <section className="grid grid-cols-12 gap-3 border-t border-(--border-color) pt-3">
        <div className="col-span-12 sm:col-span-4 sm:col-start-9">
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
