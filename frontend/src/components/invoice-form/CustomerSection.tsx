import { useEffect, useState } from "react";
import { Pencil, Plus, Repeat } from "lucide-react";
import { ApiError } from "../../lib/apiClient";
import { createCustomer, customerToPayload, listCustomers, updateCustomer } from "../../lib/customers";
import { useToast } from "../../context/ToastContext";
import type { Customer, CustomerPayload } from "../../lib/types";
import { PartyFields } from "../party-fields/PartyFields";
import { Button } from "../ui/Button";
import { Combobox } from "../ui/Combobox";

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

type Mode = "search" | "summary" | "form";

function hasData(value: CustomerPayload): boolean {
  return value.name.trim() !== "" || value.identification.number.trim() !== "";
}

// Capturar un cliente en la factura ya no muestra todos los campos de entrada: se busca por
// nombre/identificación entre los clientes guardados (Combobox) y, si no existe, se crea o
// edita desde aquí mismo — "Guardar" llama de verdad a createCustomer/updateCustomer (no es
// solo un snapshot suelto), porque si no se guarda no volvería a aparecer en la búsqueda. El
// snapshot que viaja en la factura (`value`) sigue siendo independiente del catálogo una vez
// guardado (sección 9.21/9.37) — PartyFields edita un borrador local (`draft`), no `value`
// directamente, y solo se confirma al guardar.
export function CustomerSection({ value, customerId, onChange }: CustomerSectionProps) {
  const [customers, setCustomers] = useState<Customer[]>([]);
  // Un borrador viejo con datos de cliente ya capturados pero sin customerId (snapshot suelto,
  // de antes de este cambio) arranca directo en el formulario, precargado, en vez de mostrar
  // una búsqueda vacía que esconde lo que ya había.
  const [mode, setMode] = useState<Mode>(customerId ? "summary" : hasData(value) ? "form" : "search");
  const [picker, setPicker] = useState("");
  const [draft, setDraft] = useState<CustomerPayload>(value);
  const [saving, setSaving] = useState(false);
  const toast = useToast();

  useEffect(() => {
    refreshCustomers();
  }, []);

  function refreshCustomers() {
    return listCustomers()
      .then(setCustomers)
      .catch(() => setCustomers([]));
  }

  const selectedCustomer = customers.find((c) => c.id === customerId);
  const searchOptions = customers.map((c) => ({ value: c.id, label: `${c.name} — ${c.identification.number}` }));

  function handlePick(id: string) {
    const customer = customers.find((c) => c.id === id);
    if (!customer) return;
    onChange(customerToPayload(customer), id);
    setPicker("");
    setMode("summary");
  }

  function handleChangeCustomer() {
    onChange(NEW_CUSTOMER, "");
    setMode("search");
  }

  function startCreate() {
    setDraft(NEW_CUSTOMER);
    setMode("form");
  }

  function startEdit() {
    setDraft(value);
    setMode("form");
  }

  function cancelForm() {
    setMode(customerId ? "summary" : "search");
  }

  async function handleSaveForm() {
    setSaving(true);
    try {
      const saved = customerId ? await updateCustomer(customerId, draft) : await createCustomer(draft);
      await refreshCustomers();
      onChange(customerToPayload(saved), saved.id);
      toast.success(customerId ? "Cliente actualizado." : "Cliente creado.");
      setMode("summary");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar el cliente");
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
                label="Buscar cliente"
                value={picker}
                onChange={handlePick}
                options={searchOptions}
                placeholder="Nombre o número de identificación…"
              />
            </div>
          </div>
          <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={startCreate} className="self-start">
            Crear cliente nuevo
          </Button>
        </div>
      )}

      {mode === "summary" && selectedCustomer && (
        <div className="flex items-center justify-between rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 text-xs">
          <span className="text-(--text-primary)">
            {selectedCustomer.name}
            <span className="text-(--text-secondary)"> — {selectedCustomer.identification.number}</span>
            {selectedCustomer.email && <span className="text-(--text-secondary)"> · {selectedCustomer.email}</span>}
            {selectedCustomer.phone && <span className="text-(--text-secondary)"> · {selectedCustomer.phone}</span>}
          </span>
          <div className="flex gap-1">
            <Button type="button" variant="secondary" icon={<Pencil className="h-3.5 w-3.5" />} onClick={startEdit}>
              Editar cliente
            </Button>
            <Button type="button" variant="ghost" icon={<Repeat className="h-3.5 w-3.5" />} onClick={handleChangeCustomer}>
              Cambiar cliente
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
              {customerId ? "Guardar cambios" : "Guardar cliente"}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
