import { useEffect, useState } from "react";
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
import { WithholdingTaxesEditor } from "./WithholdingTaxesEditor";
import { TotalsSummary } from "./TotalsSummary";

const SUPPORT_DOCUMENT_DIAN_TYPE = "05";

const OPERATION_TYPE_OPTIONS = [
  { code: "10", label: "Residente (10)" },
  { code: "11", label: "No Residente (11)" },
];

const NEW_VENDOR: VendorPayload = {
  identification: { number: "", type_code: "13" },
  name: "",
  tax_scheme_code: "ZZ",
  tax_regime_code: "49",
  liability_codes: ["O-49"],
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
              No hay un rango activo para Documento Soporte — créalo en Configuración → Empresa.
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

      {/* Totales alineados a la derecha */}
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
