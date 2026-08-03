import { useEffect, useState } from "react";
import { Warehouse as WarehouseIcon, Pencil, Plus, Star, Trash2 } from "lucide-react";
import { createWarehouse, deactivateWarehouse, listWarehouses, setDefaultWarehouse, updateWarehouse } from "../lib/warehouses";
import { ApiError } from "../lib/apiClient";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { Warehouse, WarehousePayload } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";

type Editing = "new" | Warehouse | null;

const EMPTY: WarehousePayload = { code: "", name: "", address: "" };

export function WarehousesPage() {
  const [warehouses, setWarehouses] = useState<Warehouse[] | null>(null);
  const [editing, setEditing] = useState<Editing>(null);
  const [form, setForm] = useState<WarehousePayload>(EMPTY);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const confirmDialog = useConfirm();
  const toast = useToast();

  function refresh() {
    listWarehouses()
      .then(setWarehouses)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar las bodegas"));
  }

  useEffect(() => { refresh(); }, []);

  function startEdit(w: Editing) {
    setEditing(w);
    setForm(w && w !== "new" ? { code: w.code, name: w.name, address: w.address } : EMPTY);
    setError(null);
  }

  async function handleSave() {
    setError(null);
    setSaving(true);
    try {
      if (editing === "new") {
        await createWarehouse(form);
        toast.success(`Bodega "${form.name}" creada.`);
      } else if (editing) {
        await updateWarehouse(editing.id, form);
        toast.success("Bodega actualizada.");
      }
      startEdit(null);
      refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar la bodega");
    } finally {
      setSaving(false);
    }
  }

  async function handleDeactivate(w: Warehouse) {
    if (!(await confirmDialog(`¿Desactivar la bodega "${w.name}"? Ya no aparecerá como destino para nuevos movimientos.`, { tone: "danger" }))) return;
    try {
      await deactivateWarehouse(w.id);
      toast.success("Bodega desactivada.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo desactivar la bodega");
    }
  }

  async function handleSetDefault(w: Warehouse) {
    try {
      await setDefaultWarehouse(w.id);
      toast.success(`"${w.name}" es ahora la bodega por defecto.`);
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo marcar como bodega por defecto");
    }
  }

  const title = editing === "new" ? "Nueva bodega" : editing ? editing.name : "Bodegas";
  const canSave = form.code.trim() !== "" && form.name.trim() !== "";

  return (
    <div className="p-4">
      <Breadcrumbs
        items={editing ? [{ label: "Inventario", to: "/inventory" }, { label: "Bodegas", onClick: () => startEdit(null) }, { label: title }] : [{ label: "Inventario", to: "/inventory" }, { label: "Bodegas" }]}
      />
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <WarehouseIcon className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          {title}
        </h1>
        {!editing && (
          <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => startEdit("new")}>
            Nueva bodega
          </Button>
        )}
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {editing ? (
        <Card className="p-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <Input label="Código" required value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} placeholder="Ej. BOG-01" />
            <Input label="Nombre" required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Ej. Bodega Bogotá" />
            <Input label="Dirección (opcional)" value={form.address} onChange={(e) => setForm({ ...form, address: e.target.value })} />
          </div>
          <div className="mt-4 flex justify-end gap-2 border-t border-(--border-light) pt-3">
            <Button type="button" variant="secondary" onClick={() => startEdit(null)}>Cancelar</Button>
            <Button type="button" disabled={!canSave} loading={saving} onClick={handleSave}>
              {editing === "new" ? "Crear bodega" : "Guardar cambios"}
            </Button>
          </div>
        </Card>
      ) : warehouses === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : warehouses.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Todavía no tienes ninguna bodega — se creará una "Principal" automáticamente en cuanto confirmes tu primera venta o compra.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Código</th>
                <th className="px-3 py-2 font-medium">Nombre</th>
                <th className="px-3 py-2 font-medium">Dirección</th>
                <th className="px-3 py-2 font-medium">Estado</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {warehouses.map((w, i) => (
                <tr key={w.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 font-mono text-(--text-primary)">{w.code}</td>
                  <td className="px-3 py-2 text-(--text-primary)">
                    <div className="flex items-center gap-1.5">
                      {w.name}
                      {w.is_default && (
                        <span className="inline-flex items-center gap-0.5 rounded bg-(--color-warning-bg) px-1.5 py-0.5 text-[10px] font-medium text-(--color-warning-text)">
                          <Star className="h-2.5 w-2.5 fill-current" /> Por defecto
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-3 py-2 text-(--text-secondary)">{w.address || "—"}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{w.is_active ? "Activa" : "Inactiva"}</td>
                  <td className="px-3 py-2">
                    <div className="flex justify-end gap-1">
                      {!w.is_default && w.is_active && (
                        <button type="button" title="Marcar por defecto" onClick={() => handleSetDefault(w)}
                          className="rounded p-1 text-(--text-secondary) transition-colors hover:bg-(--bg-hover)">
                          <Star className="h-3.5 w-3.5" />
                        </button>
                      )}
                      <button type="button" title="Editar" onClick={() => startEdit(w)}
                        className="rounded p-1 text-(--text-secondary) transition-colors hover:bg-(--bg-hover)">
                        <Pencil className="h-3.5 w-3.5" />
                      </button>
                      {w.is_active && (
                        <button type="button" title="Desactivar" onClick={() => handleDeactivate(w)}
                          className="rounded p-1 text-(--color-danger) transition-colors hover:bg-(--bg-hover)">
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      )}
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
