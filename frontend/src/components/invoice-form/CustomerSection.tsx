import { useEffect, useState } from "react";
import { customerToPayload, listCustomers } from "../../lib/customers";
import type { Customer, CustomerPayload } from "../../lib/types";
import { PartyFields } from "../party-fields/PartyFields";
import { Select } from "../ui/Select";

interface CustomerSectionProps {
  value: CustomerPayload;
  customerId: string;
  onChange: (next: CustomerPayload, customerId: string) => void;
}

const NEW_CUSTOMER: CustomerPayload = {
  identification: { number: "", type_code: "31" },
  name: "",
  tax_scheme_code: "ZZ",
  liability_codes: ["R-99-PN"],
};

// Selector de "cliente guardado" — copia TODOS los campos del Customer elegido al snapshot de
// la factura (resuelve el hallazgo de snapshot incompleto de la Fase 1, ver
// docs/frontend-architecture.md), + PartyFields editable debajo para ajustar o capturar uno
// nuevo manualmente. customerId es la referencia de trazabilidad opcional (ver
// documents.IssueInvoiceRequest.CustomerID) — se limpia sola si el usuario edita el snapshot
// después de elegir un cliente guardado, porque en ese punto ya no es exactamente ese cliente.
export function CustomerSection({ value, customerId, onChange }: CustomerSectionProps) {
  const [customers, setCustomers] = useState<Customer[]>([]);

  useEffect(() => {
    listCustomers()
      .then(setCustomers)
      .catch(() => setCustomers([]));
  }, []);

  function handleSelect(id: string) {
    if (!id) {
      onChange(NEW_CUSTOMER, "");
      return;
    }
    const customer = customers.find((c) => c.id === id);
    if (!customer) return;
    onChange(customerToPayload(customer), id);
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-12 gap-3">
        <div className="col-span-6">
          <Select label="Cliente guardado (opcional)" value={customerId} onChange={(e) => handleSelect(e.target.value)}>
            <option value="">Cliente nuevo (sin guardar)</option>
            {customers.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} — {c.identification.number}
              </option>
            ))}
          </Select>
        </div>
      </div>
      <PartyFields value={value} onChange={(next) => onChange(next, customerId)} />
    </div>
  );
}
