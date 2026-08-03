import { useEffect, useMemo, useState } from "react";
import { Pencil, Plus, Search, Trash2, Truck } from "lucide-react";
import { createSupplier, deleteSupplier, listSuppliers, updateSupplier } from "../lib/suppliers";
import { ApiError } from "../lib/apiClient";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { Supplier, SupplierPayload } from "../lib/types";
import { Card } from "../components/ui/Card";
import { Button } from "../components/ui/Button";
import { Banner } from "../components/ui/Banner";
import { Spinner } from "../components/ui/Spinner";
import { Pagination } from "../components/ui/Pagination";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { SupplierForm } from "../components/supplier-form/SupplierForm";

type Editing = "new" | Supplier | null;
const PAGE_SIZE = 10;

export function SuppliersPage() {
  const [suppliers, setSuppliers] = useState<Supplier[] | null>(null);
  const [editing, setEditing] = useState<Editing>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState("");
  const [offset, setOffset] = useState(0);
  const confirm = useConfirm();
  const toast = useToast();

  function refresh() {
    listSuppliers()
      .then(setSuppliers)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar los proveedores"));
  }

  useEffect(() => { refresh(); }, []);

  const filtered = useMemo(() => {
    if (!suppliers) return [];
    if (!search.trim()) return suppliers;
    const q = search.toLowerCase();
    return suppliers.filter(
      (s) => s.name.toLowerCase().includes(q) || s.identification_number.toLowerCase().includes(q),
    );
  }, [suppliers, search]);

  const page = filtered.slice(offset, offset + PAGE_SIZE);
  const hasNext = offset + PAGE_SIZE < filtered.length;

  function handleSearch(value: string) {
    setSearch(value);
    setOffset(0);
  }

  async function handleSave(payload: SupplierPayload) {
    setError(null);
    setLoading(true);
    try {
      if (editing === "new") {
        await createSupplier(payload);
      } else if (editing) {
        await updateSupplier(editing.id, payload);
      }
      setEditing(null);
      refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar el proveedor");
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(supplier: Supplier) {
    if (!(await confirm(`¿Eliminar a "${supplier.name}"? Esto no afecta documentos ya emitidos.`, { tone: "danger" }))) return;
    try {
      await deleteSupplier(supplier.id);
      toast.success("Proveedor eliminado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el proveedor");
    }
  }

  const title = editing === "new" ? "Nuevo proveedor" : editing ? editing.name : "Proveedores";

  return (
    <div className="p-4">
      <Breadcrumbs
        items={editing ? [{ label: "Proveedores", onClick: () => setEditing(null) }, { label: title }] : [{ label: "Proveedores" }]}
      />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Truck className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          {title}
        </h1>
        {!editing && (
          <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setEditing("new")}>
            Nuevo proveedor
          </Button>
        )}
      </div>

      {!editing && (
        <div className="mb-3 relative">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-(--text-muted)" />
          <input
            type="text"
            placeholder="Buscar por nombre o identificación…"
            value={search}
            onChange={(e) => handleSearch(e.target.value)}
            className="w-full rounded border border-(--border-color) bg-(--bg-primary) py-1.5 pl-8 pr-3 text-xs text-(--text-primary) placeholder:text-(--text-muted) transition-colors"
          />
        </div>
      )}

      {error && <Banner tone="danger">{error}</Banner>}

      {editing ? (
        <Card className="mt-3">
          <SupplierForm
            initial={editing === "new" ? null : editing}
            onSubmit={handleSave}
            onCancel={() => setEditing(null)}
            loading={loading}
          />
        </Card>
      ) : suppliers === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : filtered.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">
          {search ? "No hay proveedores que coincidan con la búsqueda." : "Todavía no tienes ningún proveedor guardado."}
        </p>
      ) : (
        <>
          <div className="overflow-x-auto rounded border border-(--border-color)">
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
                {page.map((s, i) => (
                  <tr key={s.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                    <td className="px-3 py-2 text-(--text-primary)">{s.name}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{s.identification_number}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{s.email || "—"}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{s.phone || "—"}</td>
                    <td className="px-3 py-2">
                      <div className="flex justify-end gap-1">
                        <button type="button" title="Editar" onClick={() => setEditing(s)}
                          className="rounded p-1 text-(--text-secondary) transition-colors hover:bg-(--bg-hover)">
                          <Pencil className="h-3.5 w-3.5" />
                        </button>
                        <button type="button" title="Eliminar" onClick={() => handleDelete(s)}
                          className="rounded p-1 text-(--color-danger) transition-colors hover:bg-(--bg-hover)">
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination
            offset={offset}
            count={page.length}
            hasNext={hasNext}
            onPrev={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
            onNext={() => setOffset((o) => o + PAGE_SIZE)}
          />
        </>
      )}
    </div>
  );
}
