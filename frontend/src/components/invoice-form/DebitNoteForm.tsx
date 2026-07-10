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
  IssueDebitNotePayload,
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

// DIAN — Códigos de concepto para Nota Débito (misma List 22 del Anexo Técnico, sección ND)
const DEBIT_NOTE_TYPES = [
  { code: "1", label: "Intereses" },
  { code: "2", label: "Gastos por cobrar" },
  { code: "3", label: "Cambio del valor" },
];

const DEBIT_NOTE_DIAN_TYPE = "92";

const NEW_CUSTOMER: CustomerPayload = {
  identification: { number: "", type_code: "31" },
  name: "",
  tax_scheme_code: "ZZ",
  liability_codes: ["R-99-PN"],
};

interface DebitNoteFormProps {
  initial: Document | null;
  prefill?: Document | null;
  billingReference: BillingReference;
  onSubmit: (payload: IssueDebitNotePayload) => void;
  onCancel: () => void;
  loading: boolean;
}

export function DebitNoteForm({ initial, prefill, billingReference, onSubmit, onCancel, loading }: DebitNoteFormProps) {
  const [ranges, setRanges] = useState<NumberingRange[]>([]);
  const { data: currencies, loading: loadingCurrencies } = useCatalog(listCurrencies);
  const [numberingRangeId, setNumberingRangeId] = useState(initial?.numbering_range_id ?? "");
  const [customer, setCustomer] = useState<CustomerPayload>(
    initial?.customer ?? prefill?.customer ?? NEW_CUSTOMER
  );
  const [customerId, setCustomerId] = useState(initial?.customer_id ?? prefill?.customer_id ?? "");
  const [lines, setLines] = useState<DocumentLineInput[]>(
    initial?.lines.map(lineToInput) ?? prefill?.lines.map(lineToInput) ?? []
  );
  const [paymentMeans, setPaymentMeans] = useState<PaymentMean[]>(
    initial?.payment_means ?? prefill?.payment_means ?? []
  );
  const [note, setNote] = useState(initial?.note ?? "");
  const [currencyCode, setCurrencyCode] = useState(
    initial?.currency_code ?? prefill?.currency_code ?? "COP"
  );
  const [discrepancy, setDiscrepancy] = useState<DiscrepancyResponse>(
    initial?.discrepancy_response ?? {
      reference_id: billingReference.prefix + billingReference.number,
      response_code: "",
      description: "",
    }
  );

  function handleTypeCodeChange(code: string) {
    const label = DEBIT_NOTE_TYPES.find((t) => t.code === code)?.label ?? "";
    if (!initial?.discrepancy_response) {
      setDiscrepancy((d) => ({ ...d, response_code: code, description: label }));
    }
  }

  useEffect(() => {
    listNumberingRanges(DEBIT_NOTE_DIAN_TYPE)
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
      discrepancy_response: discrepancy,
    });
  }

  const selectedRange = ranges.find((r) => r.id === numberingRangeId);
  const selectableRanges = ranges.filter((r) => r.status === "active" || r.id === numberingRangeId);
  const canSubmit =
    numberingRangeId !== "" &&
    discrepancy.response_code !== "" &&
    customer.identification.number.trim() !== "" &&
    lines.length > 0 &&
    paymentMeans.length > 0;

  return (
    <div className="flex flex-col gap-4 p-4">
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
            label="Rango de numeración"
            required
            value={numberingRangeId}
            onChange={(e) => setNumberingRangeId(e.target.value)}
          >
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
              No hay un rango de numeración activo para Nota Débito — créalo en Configuración → Empresa.
            </p>
          )}
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

      <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
        <h2 className="text-xs font-semibold text-(--text-primary)">Cliente</h2>
        <CustomerSection value={customer} customerId={customerId} onChange={handleCustomerChange} />
      </section>

      <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
        <h2 className="text-xs font-semibold text-(--text-primary)">Líneas</h2>
        <LineItemsEditor lines={lines} onChange={setLines} />
      </section>

      <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
        <h2 className="text-xs font-semibold text-(--text-primary)">Forma de pago</h2>
        <PaymentMeansEditor paymentMeans={paymentMeans} onChange={setPaymentMeans} />
      </section>

      <section className="grid grid-cols-12 gap-3 border-t border-(--border-color) pt-3">
        <div className="col-span-4 col-start-9">
          <TotalsSummary lines={lines} />
        </div>
      </section>

      <section className="flex flex-col gap-2 border-t border-(--border-color) pt-3">
        <p className="text-xs font-semibold text-(--text-primary)">Respuesta de discrepancia</p>
        <p className="text-xs text-(--text-secondary)">
          Se envía junto con la nota — se llena automáticamente al seleccionar el concepto.
        </p>
        <div className="grid grid-cols-12 gap-3">
          <div className="col-span-4">
            <Input
              label="ID de referencia (número de la factura)"
              value={discrepancy.reference_id}
              onChange={(e) => setDiscrepancy((d) => ({ ...d, reference_id: e.target.value }))}
            />
          </div>
          <div className="col-span-4">
            <Select
              label="Concepto de Nota Débito"
              required
              value={discrepancy.response_code}
              onChange={(e) => {
                handleTypeCodeChange(e.target.value);
                setDiscrepancy((d) => ({ ...d, response_code: e.target.value }));
              }}
            >
              <option value="">Selecciona…</option>
              {DEBIT_NOTE_TYPES.map((t) => (
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
