import { useEffect, useState } from "react";
import { Pencil, Plus, Repeat } from "lucide-react";
import { ApiError } from "../../lib/apiClient";
import { createVendor, listVendors, updateVendor, vendorToPayload } from "../../lib/vendors";
import { useToast } from "../../context/ToastContext";
import type { Vendor, VendorPayload } from "../../lib/types";
import { PartyFields } from "../party-fields/PartyFields";
import { Button } from "../ui/Button";
import { Combobox } from "../ui/Combobox";

interface VendorSectionProps {
  value: VendorPayload;
  vendorId: string;
  onChange: (next: VendorPayload, vendorId: string) => void;
}

const NEW_VENDOR: VendorPayload = {
  identification: { number: "", type_code: "13" },
  name: "",
  entity_type_code: "1",
  tax_scheme_code: "ZZ",
  liability_codes: ["R-99-PN"],
  tax_regime_code: "",
};

type Mode = "search" | "summary" | "form";

function hasData(value: VendorPayload): boolean {
  return value.name.trim() !== "" || value.identification.number.trim() !== "";
}

export function VendorSection({ value, vendorId, onChange }: VendorSectionProps) {
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [loadingVendors, setLoadingVendors] = useState(true);
  const [mode, setMode] = useState<Mode>(vendorId ? "summary" : hasData(value) ? "form" : "search");
  const [picker, setPicker] = useState("");
  const [draft, setDraft] = useState<VendorPayload>(value);
  const [saving, setSaving] = useState(false);
  const toast = useToast();

  useEffect(() => {
    refreshVendors();
  }, []);

  function refreshVendors() {
    return listVendors()
      .then(setVendors)
      .catch(() => setVendors([]))
      .finally(() => setLoadingVendors(false));
  }

  const selectedVendor = vendors.find((v) => v.id === vendorId);
  const searchOptions = vendors.map((v) => ({ value: v.id, label: `${v.name} — ${v.identification.number}` }));

  function handlePick(id: string) {
    const vendor = vendors.find((v) => v.id === id);
    if (!vendor) return;
    onChange(vendorToPayload(vendor), id);
    setPicker("");
    setMode("summary");
  }

  function handleChangeVendor() {
    onChange(NEW_VENDOR, "");
    setMode("search");
  }

  function startCreate() {
    setDraft(NEW_VENDOR);
    setMode("form");
  }

  function startEdit() {
    setDraft(value);
    setMode("form");
  }

  function cancelForm() {
    setMode(vendorId ? "summary" : "search");
  }

  async function handleSaveForm() {
    setSaving(true);
    try {
      const saved = vendorId ? await updateVendor(vendorId, draft) : await createVendor(draft);
      await refreshVendors();
      onChange(vendorToPayload(saved), saved.id);
      toast.success(vendorId ? "Proveedor actualizado." : "Proveedor creado.");
      setMode("summary");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar el proveedor");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      {mode === "search" && (
        <div className="flex flex-col gap-2">
          <div className="grid grid-cols-12 gap-3">
            <div className="col-span-6">
              <Combobox
                label="Buscar proveedor"
                value={picker}
                onChange={handlePick}
                options={searchOptions}
                placeholder={loadingVendors ? "Cargando proveedores…" : "Nombre o número de identificación…"}
                disabled={loadingVendors}
              />
            </div>
          </div>
          <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={startCreate} className="self-start">
            Crear proveedor nuevo
          </Button>
        </div>
      )}

      {mode === "summary" && selectedVendor && (
        <div className="flex items-center justify-between rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 text-xs">
          <span className="text-(--text-primary)">
            {selectedVendor.name}
            <span className="text-(--text-secondary)"> — {selectedVendor.identification.number}</span>
            {selectedVendor.email && <span className="text-(--text-secondary)"> · {selectedVendor.email}</span>}
            {selectedVendor.phone && <span className="text-(--text-secondary)"> · {selectedVendor.phone}</span>}
          </span>
          <div className="flex gap-1">
            <Button type="button" variant="secondary" icon={<Pencil className="h-3.5 w-3.5" />} onClick={startEdit}>
              Editar proveedor
            </Button>
            <Button type="button" variant="ghost" icon={<Repeat className="h-3.5 w-3.5" />} onClick={handleChangeVendor}>
              Cambiar proveedor
            </Button>
          </div>
        </div>
      )}

      {mode === "form" && (
        <div className="flex flex-col gap-3">
          <PartyFields value={draft} onChange={setDraft} />
          <div className="flex gap-2">
            <Button type="button" variant="secondary" onClick={cancelForm}>
              Cancelar
            </Button>
            <Button type="button" loading={saving} onClick={handleSaveForm}>
              {vendorId ? "Guardar cambios" : "Guardar proveedor"}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
