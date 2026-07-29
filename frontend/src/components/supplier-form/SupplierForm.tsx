import { useState, type FormEvent } from "react";
import { supplierToPayload } from "../../lib/suppliers";
import type { Supplier, SupplierPayload } from "../../lib/types";
import { PartyFields } from "../party-fields/PartyFields";
import { Button } from "../ui/Button";

interface SupplierFormProps {
  initial: Supplier | null;
  onSubmit: (payload: SupplierPayload) => void;
  onCancel: () => void;
  loading: boolean;
}

function payloadFromSupplier(supplier: Supplier | null): SupplierPayload {
  if (!supplier) {
    return {
      identification: { number: "", type_code: "13" },
      name: "",
      tax_scheme_code: "ZZ",
      tax_regime_code: "49",
      liability_codes: ["O-49"],
    };
  }
  return supplierToPayload(supplier);
}

export function SupplierForm({ initial, onSubmit, onCancel, loading }: SupplierFormProps) {
  const [payload, setPayload] = useState<SupplierPayload>(() => payloadFromSupplier(initial));

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    onSubmit(payload);
  }

  return (
    <form className="flex flex-col gap-3 p-4" onSubmit={handleSubmit}>
      <PartyFields value={payload} onChange={setPayload} />

      <div className="flex gap-2">
        <Button type="button" variant="secondary" onClick={onCancel} className="flex-1">
          Cancelar
        </Button>
        <Button type="submit" loading={loading} className="flex-1">
          {initial ? "Guardar cambios" : "Crear proveedor"}
        </Button>
      </div>
    </form>
  );
}
