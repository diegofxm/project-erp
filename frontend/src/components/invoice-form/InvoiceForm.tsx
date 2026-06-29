import { useEffect, useState } from "react";
import { listCurrencies } from "../../lib/catalogs";
import { lineToInput } from "../../lib/documents";
import { listNumberingRanges } from "../../lib/numberingRanges";
import { useCatalog } from "../../lib/useCatalog";
import type { CustomerPayload, Document, DocumentLineInput, IssueInvoicePayload, NumberingRange, PaymentMean } from "../../lib/types";
import { Button } from "../ui/Button";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { CustomerSection } from "./CustomerSection";
import { LineItemsEditor } from "./LineItemsEditor";
import { PaymentMeansEditor } from "./PaymentMeansEditor";
import { TotalsSummary } from "./TotalsSummary";

interface InvoiceFormProps {
  initial: Document | null; // null = factura nueva
  onSubmit: (payload: IssueInvoicePayload) => void;
  onCancel: () => void;
  loading: boolean;
}

const NEW_CUSTOMER: CustomerPayload = {
  identification: { number: "", type_code: "31" },
  name: "",
  tax_scheme_code: "ZZ",
  liability_codes: ["R-99-PN"],
};

const INVOICE_DIAN_DOCUMENT_TYPE = "01";

export function InvoiceForm({ initial, onSubmit, onCancel, loading }: InvoiceFormProps) {
  const [ranges, setRanges] = useState<NumberingRange[]>([]);
  const { data: currencies, loading: loadingCurrencies } = useCatalog(listCurrencies);
  const [numberingRangeId, setNumberingRangeId] = useState(initial?.numbering_range_id ?? "");
  const [customer, setCustomer] = useState<CustomerPayload>(initial?.customer ?? NEW_CUSTOMER);
  const [customerId, setCustomerId] = useState(initial?.customer_id ?? "");
  const [lines, setLines] = useState<DocumentLineInput[]>(initial?.lines.map(lineToInput) ?? []);
  const [paymentMeans, setPaymentMeans] = useState<PaymentMean[]>(initial?.payment_means ?? []);
  const [note, setNote] = useState(initial?.note ?? "");
  const [currencyCode, setCurrencyCode] = useState(initial?.currency_code ?? "COP");

  useEffect(() => {
    listNumberingRanges(INVOICE_DIAN_DOCUMENT_TYPE)
      .then(setRanges)
      .catch(() => setRanges([]));
  }, []);

  function handleCustomerChange(next: CustomerPayload, nextCustomerId: string) {
    setCustomer(next);
    setCustomerId(nextCustomerId);
  }

  function handleSubmit() {
    onSubmit({
      numbering_range_id: numberingRangeId,
      customer,
      lines,
      payment_means: paymentMeans.length > 0 ? paymentMeans : undefined,
      note: note || undefined,
      currency_code: currencyCode,
      customer_id: customerId || undefined,
    });
  }

  const selectedRange = ranges.find((r) => r.id === numberingRangeId);
  // Solo se ofrecen rangos activos (ver numbering.NumberingRange.Status en apidian) — si el
  // rango de un borrador ya en edición venció/se agotó mientras tanto, igual se mantiene
  // visible (es el seleccionado), para no dejar el select con un valor huérfano.
  const selectableRanges = ranges.filter((r) => r.status === "active" || r.id === numberingRangeId);
  const canSubmit =
    numberingRangeId !== "" && customer.identification.number.trim() !== "" && lines.length > 0 && paymentMeans.length > 0;

  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="grid grid-cols-12 gap-3">
        <div className="col-span-6">
          <Select label="Rango de numeración" required value={numberingRangeId} onChange={(e) => setNumberingRangeId(e.target.value)}>
            <option value="">Selecciona…</option>
            {selectableRanges.map((r) => (
              <option key={r.id} value={r.id}>
                {r.prefix}
              </option>
            ))}
          </Select>
          {selectedRange && (
            <p className="mt-1 text-xs text-(--text-muted)">Próximo número: {selectedRange.prefix}{selectedRange.current_number + 1}</p>
          )}
          {selectableRanges.length === 0 && (
            <p className="mt-1 text-xs text-(--text-muted)">
              No hay un rango de numeración activo para Factura Electrónica todavía — créalo en Configuración → Empresa.
            </p>
          )}
        </div>
        <div className="col-span-3">
          <Select label="Moneda" disabled={loadingCurrencies} value={currencyCode} onChange={(e) => setCurrencyCode(e.target.value)}>
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

      <section className="flex flex-col gap-2">
        <h2 className="text-xs font-semibold text-(--text-primary)">Cliente</h2>
        <CustomerSection value={customer} customerId={customerId} onChange={handleCustomerChange} />
      </section>

      <section className="flex flex-col gap-2">
        <h2 className="text-xs font-semibold text-(--text-primary)">Líneas</h2>
        <LineItemsEditor lines={lines} onChange={setLines} />
      </section>

      <section className="flex flex-col gap-2">
        <h2 className="text-xs font-semibold text-(--text-primary)">Forma de pago</h2>
        <PaymentMeansEditor paymentMeans={paymentMeans} onChange={setPaymentMeans} />
      </section>

      <section className="grid grid-cols-12 gap-3">
        <div className="col-span-4 col-start-9">
          <TotalsSummary lines={lines} />
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
