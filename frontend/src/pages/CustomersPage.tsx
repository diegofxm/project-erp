import { useEffect, useState } from "react";
import { Pencil, Plus, Trash2, Users } from "lucide-react";
import { createCustomer, deleteCustomer, listCustomers, updateCustomer } from "../lib/customers";
import { ApiError } from "../lib/apiClient";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { Customer, CustomerPayload } from "../lib/types";
import { Card } from "../components/ui/Card";
import { Button } from "../components/ui/Button";
import { Banner } from "../components/ui/Banner";
import { Spinner } from "../components/ui/Spinner";
import { CustomerForm } from "../components/customer-form/CustomerForm";

type Editing = "new" | Customer | null;

// Listado a todo el ancho (ver docs/frontend-architecture.md, regla de ancho) — el formulario
// de creación/edición reemplaza la tabla mientras está abierto, mismo patrón que
// IssuerManager/NumberingRangesPanel.
export function CustomersPage() {
  const [customers, setCustomers] = useState<Customer[] | null>(null);
  const [editing, setEditing] = useState<Editing>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const confirm = useConfirm();
  const toast = useToast();

  function refresh() {
    listCustomers()
      .then(setCustomers)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar los clientes"));
  }

  useEffect(() => {
    refresh();
  }, []);

  async function handleSave(payload: CustomerPayload) {
    setError(null);
    setLoading(true);
    try {
      if (editing === "new") {
        await createCustomer(payload);
      } else if (editing) {
        await updateCustomer(editing.id, payload);
      }
      setEditing(null);
      refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar el cliente");
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(customer: Customer) {
    if (!(await confirm(`¿Eliminar a "${customer.name}"? Esto no afecta documentos ya emitidos.`, { tone: "danger" }))) return;
    try {
      await deleteCustomer(customer.id);
      toast.success("Cliente eliminado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el cliente");
    }
  }

  return (
    <div className="p-4">
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
            <Users className="h-4 w-4 shrink-0 text-(--accent-primary)" />
            Clientes
          </h1>
        {!editing && (
          <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setEditing("new")}>
            Nuevo cliente
          </Button>
        )}
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {editing ? (
        <Card className="mt-3">
          <CustomerForm
            initial={editing === "new" ? null : editing}
            onSubmit={handleSave}
            onCancel={() => setEditing(null)}
            loading={loading}
          />
        </Card>
      ) : customers === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : customers.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Todavía no tienes ningún cliente guardado.</p>
      ) : (
        <div className="overflow-hidden rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Nombre</th>
                <th className="px-3 py-2 font-medium">Identificación</th>
                <th className="px-3 py-2 font-medium">Correo</th>
                <th className="px-3 py-2 font-medium">Teléfono</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {customers.map((c, i) => (
                <tr key={c.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 text-(--text-primary)">{c.name}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{c.identification.number}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{c.email || "—"}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{c.phone || "—"}</td>
                  <td className="px-3 py-2">
                    <div className="flex justify-end gap-1">
                      <button
                        type="button"
                        title="Editar"
                        onClick={() => setEditing(c)}
                        className="rounded p-1 text-(--text-secondary) hover:bg-(--bg-hover)"
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </button>
                      <button
                        type="button"
                        title="Eliminar"
                        onClick={() => handleDelete(c)}
                        className="rounded p-1 text-(--color-danger) hover:bg-(--bg-hover)"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
