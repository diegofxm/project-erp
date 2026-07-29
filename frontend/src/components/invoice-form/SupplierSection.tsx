import { useEffect, useState } from "react";
import { Pencil, Plus, Repeat } from "lucide-react";
import { ApiError } from "../../lib/apiClient";
import { createSupplier, listSuppliers, updateSupplier, supplierToPayload } from "../../lib/suppliers";
import { useToast } from "../../context/ToastContext";
import type { Supplier, SupplierPayload } from "../../lib/types";
import { PartyFields } from "../party-fields/PartyFields";
import { Button } from "../ui/Button";
import { Combobox } from "../ui/Combobox";

interface SupplierSectionProps {
  value: SupplierPayload;
  supplierId: string;
  onChange: (next: SupplierPayload, supplierId: string) => void;
}

const NEW_SUPPLIER: SupplierPayload = {
  identification: { number: "", type_code: "13" },
  name: "",
  tax_scheme_code: "ZZ",
  tax_regime_code: "49",
  liability_codes: ["O-49"],
};

type Mode = "search" | "summary" | "form";

function hasData(value: SupplierPayload): boolean {
  return value.name.trim() !== "" || value.identification.number.trim() !== "";
}

export function SupplierSection({ value, supplierId, onChange }: SupplierSectionProps) {
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [loadingSuppliers, setLoadingSuppliers] = useState(true);
  const [mode, setMode] = useState<Mode>(supplierId ? "summary" : hasData(value) ? "form" : "search");
  const [picker, setPicker] = useState("");
  const [draft, setDraft] = useState<SupplierPayload>(value);
  const [saving, setSaving] = useState(false);
  const toast = useToast();

  useEffect(() => {
    refreshSuppliers();
  }, []);

  function refreshSuppliers() {
    return listSuppliers()
      .then(setSuppliers)
      .catch(() => setSuppliers([]))
      .finally(() => setLoadingSuppliers(false));
  }

  const selectedSupplier = suppliers.find((s) => s.id === supplierId);
  const searchOptions = suppliers.map((s) => ({ value: s.id, label: `${s.name} — ${s.identification_number}` }));

  function handlePick(id: string) {
    const supplier = suppliers.find((s) => s.id === id);
    if (!supplier) return;
    onChange(supplierToPayload(supplier), id);
    setPicker("");
    setMode("summary");
  }

  function handleChangeSupplier() {
    onChange(NEW_SUPPLIER, "");
    setMode("search");
  }

  function startCreate() {
    setDraft(NEW_SUPPLIER);
    setMode("form");
  }

  function startEdit() {
    setDraft(value);
    setMode("form");
  }

  function cancelForm() {
    setMode(supplierId ? "summary" : "search");
  }

  async function handleSaveForm() {
    setSaving(true);
    try {
      const saved = supplierId ? await updateSupplier(supplierId, draft) : await createSupplier(draft);
      await refreshSuppliers();
      onChange(supplierToPayload(saved), saved.id);
      toast.success(supplierId ? "Proveedor actualizado." : "Proveedor creado.");
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
            <div className="col-span-12 sm:col-span-6">
              <Combobox
                label="Buscar proveedor"
                value={picker}
                onChange={handlePick}
                options={searchOptions}
                placeholder={loadingSuppliers ? "Cargando proveedores…" : "Nombre o número de identificación…"}
                disabled={loadingSuppliers}
              />
            </div>
          </div>
          <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={startCreate} className="self-start">
            Crear proveedor nuevo
          </Button>
        </div>
      )}

      {mode === "summary" && selectedSupplier && (
        <div className="flex items-center justify-between rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 text-xs">
          <span className="text-(--text-primary)">
            {selectedSupplier.name}
            <span className="text-(--text-secondary)"> — {selectedSupplier.identification_number}</span>
            {selectedSupplier.email && <span className="text-(--text-secondary)"> · {selectedSupplier.email}</span>}
            {selectedSupplier.phone && <span className="text-(--text-secondary)"> · {selectedSupplier.phone}</span>}
          </span>
          <div className="flex gap-1">
            <Button type="button" variant="secondary" icon={<Pencil className="h-3.5 w-3.5" />} onClick={startEdit}>
              Editar proveedor
            </Button>
            <Button type="button" variant="ghost" icon={<Repeat className="h-3.5 w-3.5" />} onClick={handleChangeSupplier}>
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
              {supplierId ? "Guardar cambios" : "Guardar proveedor"}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
