import { useEffect, useState } from "react";
import { listCurrencies } from "../../lib/catalogs";
import { lineToInput } from "../../lib/documents";
import { previewTotals } from "../../lib/invoiceMath";
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
  // prefill: datos de la factura de origen — solo se usa cuando initial es null (borrador nuevo).
  // Cuando initial existe (edición de borrador ya guardado), initial tiene prioridad.
  prefill?: Document | null;
  billingReference: BillingReference;
  onSubmit: (payload: IssueCreditNotePayload) => void;
  onCancel: () => void;
  loading: boolean;
  onTotalChange?: (cents: number) => void;
}

export function CreditNoteForm({ initial, prefill, billingReference, onSubmit, onCancel, loading, onTotalChange }: CreditNoteFormProps) {
  const [ranges, setRanges] = useState<NumberingRange[]>([]);
  const [loadingRanges, setLoadingRanges] = useState(true);
  const { data: currencies, loading: loadingCurrencies } = useCatalog(listCurrencies);
  const [numberingRangeId, setNumberingRangeId] = useState(initial?.numbering_range_id ?? "");
  // Cuando es un borrador nuevo (initial null), pre-llena desde la factura origen (prefill) para
  // que el usuario no tenga que re-capturar cliente e ítems desde cero — puede modificarlos si
  // la nota es parcial.
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
  const [creditNoteTypeCode, setCreditNoteTypeCode] = useState(initial?.note_type_code ?? "");
  // DiscrepancyResponse se auto-popula cuando el usuario elige el concepto — siempre se incluye
  // porque el test real de habilitación la trae y la DIAN la espera aunque sea opcional en el
  // esquema UBL (ver cofacture/builder/notes.go y realsend_creditnote_test.go).
  const [discrepancy, setDiscrepancy] = useState<DiscrepancyResponse>(
    initial?.discrepancy_response ?? {
      reference_id: billingReference.prefix + billingReference.number,
      response_code: "",
      description: "",
    }
  );

  // Al cambiar el concepto de la NC, sincroniza automáticamente la DiscrepancyResponse para
  // que el usuario no tenga que seleccionar el mismo código dos veces ni escribir la descripción
  // a mano — ReferenceID viene del billingReference (prefijo+número de la factura), ResponseCode
  // es el mismo código de la Lista 22.
  function handleTypeCodeChange(code: string) {
    setCreditNoteTypeCode(code);
    if (!initial?.discrepancy_response) {
      const label = CREDIT_NOTE_TYPES.find((t) => t.code === code)?.label ?? "";
      setDiscrepancy((d) => ({ ...d, response_code: code, description: label }));
    }
  }

  useEffect(() => {
    listNumberingRanges(CREDIT_NOTE_DIAN_TYPE)
      .then(setRanges)
      .catch(() => setRanges([]))
      .finally(() => setLoadingRanges(false));
  }, []);

  useEffect(() => {
    onTotalChange?.(previewTotals(lines).payableCents);
  }, [lines]); // eslint-disable-line react-hooks/exhaustive-deps

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
      // DiscrepancyResponse siempre se incluye — el test real de habilitación la exige aunque
      // el esquema UBL la declare opcional.
      discrepancy_response: discrepancy,
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
      <div className="grid grid-cols-12 gap-3">
        <div className="col-span-12 sm:col-span-6">
          <Select
            label="Concepto de Nota Crédito"
            required
            value={creditNoteTypeCode}
            onChange={(e) => handleTypeCodeChange(e.target.value)}
          >
            <option value="">Selecciona…</option>
            {CREDIT_NOTE_TYPES.map((t) => (
              <option key={t.code} value={t.code}>
                {t.code} — {t.label}
              </option>
            ))}
          </Select>
        </div>
        <div className="col-span-12 sm:col-span-6">
          <Select label="Rango de numeración" required value={numberingRangeId} onChange={(e) => setNumberingRangeId(e.target.value)} disabled={loadingRanges}>
            {loadingRanges ? (
              <option>Cargando…</option>
            ) : (
              <>
                <option value="">Selecciona…</option>
                {selectableRanges.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.prefix}
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
          {selectableRanges.length === 0 && (
            <p className="mt-1 text-xs text-(--text-muted)">
              No hay un rango de numeración activo para Nota Crédito — créalo en Configuración → Empresa.
            </p>
          )}
        </div>
        <div className="col-span-6 sm:col-span-3">
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
        <div className="col-span-12 sm:col-span-4 sm:col-start-9">
          <TotalsSummary lines={lines} />
        </div>
      </section>

      {/* DiscrepancyResponse — solo confirmación visual, no editable */}
      {discrepancy.response_code && (
        <section className="flex flex-col gap-1.5 border-t border-(--border-color) pt-3">
          <p className="text-xs font-semibold text-(--text-primary)">Respuesta de discrepancia</p>
          <div className="rounded border border-(--border-color) bg-(--bg-subtle) px-3 py-2 text-xs text-(--text-secondary)">
            <span className="font-medium text-(--text-primary)">{discrepancy.reference_id}</span>
            {" · "}
            <span>{discrepancy.response_code} — {discrepancy.description}</span>
          </div>
        </section>
      )}

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
