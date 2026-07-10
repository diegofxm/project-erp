import { useEffect, useState } from "react";
import { listCurrencies } from "../../lib/catalogs";
import { lineToInput } from "../../lib/documents";
import { listNumberingRanges } from "../../lib/numberingRanges";
import { useCatalog } from "../../lib/useCatalog";
import type {
  BillingReference,
  CustomerPayload,
  DiscrepancyResponse,
  Document,
  DocumentLineInput,
  IssueCreditNotePayload,
  NumberingRange,
  PaymentMean,
} from "../../lib/types";
import { Button } from "../ui/Button";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { CustomerSection } from "./CustomerSection";
import { LineItemsEditor } from "./LineItemsEditor";
import { PaymentMeansEditor } from "./PaymentMeansEditor";
import { TotalsSummary } from "./TotalsSummary";

// DIAN List 22 — Código de Concepto de Nota Crédito (Anexo Técnico v2.1, tabla 13.3.22)
const CREDIT_NOTE_TYPES = [
  { code: "1", label: "Devolución parcial de los bienes; no aceptación de partes del servicio" },
  { code: "2", label: "Anulación de factura electrónica" },
  { code: "3", label: "Rebaja o descuento parcial" },
  { code: "4", label: "Ajuste de precio" },
  { code: "5", label: "Otros" },
];

const CREDIT_NOTE_DIAN_TYPE = "91";

const NEW_CUSTOMER: CustomerPayload = {
  identification: { number: "", type_code: "31" },
  name: "",
  tax_scheme_code: "ZZ",
  liability_codes: ["R-99-PN"],
};

interface CreditNoteFormProps {
  initial: Document | null;
  billingReference: BillingReference;
  onSubmit: (payload: IssueCreditNotePayload) => void;
  onCancel: () => void;
  loading: boolean;
}

export function CreditNoteForm({ initial, billingReference, onSubmit, onCancel, loading }: CreditNoteFormProps) {
  const [ranges, setRanges] = useState<NumberingRange[]>([]);
  const { data: currencies, loading: loadingCurrencies } = useCatalog(listCurrencies);
  const [numberingRangeId, setNumberingRangeId] = useState(initial?.numbering_range_id ?? "");
  const [customer, setCustomer] = useState<CustomerPayload>(initial?.customer ?? NEW_CUSTOMER);
  const [customerId, setCustomerId] = useState(initial?.customer_id ?? "");
  const [lines, setLines] = useState<DocumentLineInput[]>(initial?.lines.map(lineToInput) ?? []);
  const [paymentMeans, setPaymentMeans] = useState<PaymentMean[]>(initial?.payment_means ?? []);
  const [note, setNote] = useState(initial?.note ?? "");
  const [currencyCode, setCurrencyCode] = useState(initial?.currency_code ?? "COP");
  const [creditNoteTypeCode, setCreditNoteTypeCode] = useState(initial?.note_type_code ?? "");
  const [hasDiscrepancy, setHasDiscrepancy] = useState(!!initial?.discrepancy_response);
  const [discrepancy, setDiscrepancy] = useState<DiscrepancyResponse>(
    initial?.discrepancy_response ?? { reference_id: "", response_code: "", description: "" }
  );

  useEffect(() => {
    listNumberingRanges(CREDIT_NOTE_DIAN_TYPE)
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
      billing_reference: billingReference,
      credit_note_type_code: creditNoteTypeCode,
      discrepancy_response: hasDiscrepancy ? discrepancy : undefined,
    });
  }

  const selectedRange = ranges.find((r) => r.id === numberingRangeId);
  const selectableRanges = ranges.filter((r) => r.status === "active" || r.id === numberingRangeId);
  const canSubmit =
    numberingRangeId !== "" &&
    creditNoteTypeCode !== "" &&
    customer.identification.number.trim() !== "" &&
    lines.length > 0 &&
    paymentMeans.length > 0;

  return (
    <div className="flex flex-col gap-4 p-4">
      {/* Referencia a la factura — solo lectura, no se puede cambiar */}
      <div className="rounded border border-(--border-color) bg-(--bg-secondary) p-3">
        <p className="text-xs font-medium text-(--text-secondary)">Factura de referencia</p>
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

      <div className="grid grid-cols-12 gap-3">
        <div className="col-span-6">
          <Select
            label="Concepto de Nota Crédito"
            required
            value={creditNoteTypeCode}
            onChange={(e) => setCreditNoteTypeCode(e.target.value)}
          >
            <option value="">Selecciona…</option>
            {CREDIT_NOTE_TYPES.map((t) => (
              <option key={t.code} value={t.code}>
                {t.code} — {t.label}
              </option>
            ))}
          </Select>
        </div>
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
            <p className="mt-1 text-xs text-(--text-muted)">
              Próximo número: {selectedRange.prefix}{selectedRange.current_number + 1}
            </p>
          )}
          {selectableRanges.length === 0 && (
            <p className="mt-1 text-xs text-(--text-muted)">
              No hay un rango de numeración activo para Nota Crédito — créalo en Configuración → Empresa.
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

      {/* DiscrepancyResponse — opcional según el anexo técnico */}
      <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
        <label className="flex cursor-pointer select-none items-center gap-2 text-xs text-(--text-secondary)">
          <input
            type="checkbox"
            checked={hasDiscrepancy}
            onChange={(e) => setHasDiscrepancy(e.target.checked)}
            className="rounded"
          />
          Incluir respuesta de discrepancia (opcional)
        </label>
        {hasDiscrepancy && (
          <div className="grid grid-cols-12 gap-3 pl-4">
            <div className="col-span-4">
              <Input
                label="ID de referencia"
                value={discrepancy.reference_id}
                onChange={(e) => setDiscrepancy((d) => ({ ...d, reference_id: e.target.value }))}
              />
            </div>
            <div className="col-span-4">
              <Select
                label="Código de respuesta"
                value={discrepancy.response_code}
                onChange={(e) => setDiscrepancy((d) => ({ ...d, response_code: e.target.value }))}
              >
                <option value="">Selecciona…</option>
                {CREDIT_NOTE_TYPES.map((t) => (
                  <option key={t.code} value={t.code}>
                    {t.code} — {t.label}
                  </option>
                ))}
              </Select>
            </div>
            <div className="col-span-12">
              <Input
                label="Descripción"
                value={discrepancy.description}
                onChange={(e) => setDiscrepancy((d) => ({ ...d, description: e.target.value }))}
              />
            </div>
          </div>
        )}
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
