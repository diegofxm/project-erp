import { useState, type FormEvent } from "react";
import { customerToPayload } from "../../lib/customers";
import type { Customer, CustomerPayload } from "../../lib/types";
import { PartyFields } from "../party-fields/PartyFields";
import { Button } from "../ui/Button";
import { Input } from "../ui/Input";


interface CustomerFormProps {
  initial: Customer | null;
  onSubmit: (payload: CustomerPayload) => void;
  onCancel: () => void;
  loading: boolean;
}

function payloadFromCustomer(customer: Customer | null): CustomerPayload {
  if (!customer) {
    return {
      identification: { number: "", type_code: "31" },
      name: "",
      tax_scheme_code: "ZZ",
      liability_codes: ["R-99-PN"],
    };
  }
  return customerToPayload(customer);
}

// Formulario único (sin pestañas, a diferencia de CompanyForm): un cliente tiene menos campos
// que una empresa — sin software/certificado, sin CIIU (cofacture/domain/types.go: los códigos
// CIIU "solo aplican al emisor, nunca al receptor"), sin matrícula mercantil ("siempre nil para
// el receptor"). Los campos en sí viven en PartyFields (compartido con la sección de cliente
// de Factura Electrónica) — este componente solo aporta el estado y el chrome de guardar.
export function CustomerForm({ initial, onSubmit, onCancel, loading }: CustomerFormProps) {
  const [payload, setPayload] = useState<CustomerPayload>(() => payloadFromCustomer(initial));

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    onSubmit(payload);
  }

  return (
    <form className="flex flex-col gap-3 p-4" onSubmit={handleSubmit}>
      <PartyFields value={payload} onChange={setPayload} />

      <Input
        label="Cupo de crédito (COP, opcional)"
        type="number"
        min="0"
        step="1"
        value={payload.credit_limit ?? ""}
        onChange={(e) => setPayload({ ...payload, credit_limit: e.target.value === "" ? null : Number(e.target.value) })}
        placeholder="Vacío = sin límite"
      />

      <div className="flex gap-2">
        <Button type="button" variant="secondary" onClick={onCancel} className="flex-1">
          Cancelar
        </Button>
        <Button type="submit" loading={loading} className="flex-1">
          {initial ? "Guardar cambios" : "Crear cliente"}
        </Button>
      </div>
    </form>
  );
}
