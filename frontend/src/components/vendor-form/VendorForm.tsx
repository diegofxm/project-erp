import { useState, type FormEvent } from "react";
import { vendorToPayload } from "../../lib/vendors";
import type { Vendor, VendorPayload } from "../../lib/types";
import { PartyFields } from "../party-fields/PartyFields";
import { Button } from "../ui/Button";

interface VendorFormProps {
  initial: Vendor | null;
  onSubmit: (payload: VendorPayload) => void;
  onCancel: () => void;
  loading: boolean;
}

function payloadFromVendor(vendor: Vendor | null): VendorPayload {
  if (!vendor) {
    return {
      identification: { number: "", type_code: "13" },
      name: "",
      tax_scheme_code: "ZZ",
      liability_codes: ["R-99-PN"],
    };
  }
  return vendorToPayload(vendor);
}

export function VendorForm({ initial, onSubmit, onCancel, loading }: VendorFormProps) {
  const [payload, setPayload] = useState<VendorPayload>(() => payloadFromVendor(initial));

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
