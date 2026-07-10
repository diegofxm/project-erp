import { useEffect, useState } from "react";
import { Package, Pencil, Plus, Trash2 } from "lucide-react";
import { createProduct, deleteProduct, listProducts, updateProduct } from "../lib/products";
import { ApiError } from "../lib/apiClient";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { Product, ProductPayload } from "../lib/types";
import { Card } from "../components/ui/Card";
import { Button } from "../components/ui/Button";
import { Banner } from "../components/ui/Banner";
import { Spinner } from "../components/ui/Spinner";
import { ProductForm } from "../components/product-form/ProductForm";

type Editing = "new" | Product | null;

const formatCOP = new Intl.NumberFormat("es-CO", { style: "currency", currency: "COP", maximumFractionDigits: 2 });

// Listado a todo el ancho (ver docs/frontend-architecture.md, regla de ancho) — mismo patrón
// que CustomersPage: el formulario reemplaza la tabla mientras está abierto.
export function ProductsPage() {
  const [products, setProducts] = useState<Product[] | null>(null);
  const [editing, setEditing] = useState<Editing>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const confirm = useConfirm();
  const toast = useToast();

  function refresh() {
    listProducts()
      .then(setProducts)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar los productos"));
  }

  useEffect(() => {
    refresh();
  }, []);

  async function handleSave(payload: ProductPayload) {
    setError(null);
    setLoading(true);
    try {
      if (editing === "new") {
        await createProduct(payload);
      } else if (editing) {
        await updateProduct(editing.id, payload);
      }
      setEditing(null);
      refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar el producto");
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(product: Product) {
    if (!(await confirm(`¿Eliminar "${product.description}"? Esto no afecta documentos ya emitidos.`, { tone: "danger" }))) return;
    try {
      await deleteProduct(product.id);
      toast.success("Producto eliminado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el producto");
    }
  }

  return (
    <div className="p-4">
      <div className="mb-3 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
            <Package className="h-4 w-4 shrink-0 text-(--accent-primary)" />
            Productos
          </h1>
        {!editing && (
          <Button type="button" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setEditing("new")}>
            Nuevo producto
          </Button>
        )}
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {editing ? (
        <Card className="mt-3">
          <ProductForm
            initial={editing === "new" ? null : editing}
            onSubmit={handleSave}
            onCancel={() => setEditing(null)}
            loading={loading}
          />
        </Card>
      ) : products === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : products.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Todavía no tienes ningún producto guardado.</p>
      ) : (
        <div className="overflow-hidden rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Descripción</th>
                <th className="px-3 py-2 font-medium">Unidad</th>
                <th className="px-3 py-2 font-medium">Precio unitario</th>
                <th className="px-3 py-2 font-medium">Impuesto</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {products.map((p, i) => (
                <tr key={p.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 text-(--text-primary)">{p.description}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{p.unit_code}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(p.unit_price_cents / 100)}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">
                    {p.tax_type_code ? `${p.tax_type_code} (${p.tax_percent ?? 0}%)` : "—"}
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex justify-end gap-1">
                      <button
                        type="button"
                        title="Editar"
                        onClick={() => setEditing(p)}
                        className="rounded p-1 text-(--text-secondary) hover:bg-(--bg-hover)"
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </button>
                      <button
                        type="button"
                        title="Eliminar"
                        onClick={() => handleDelete(p)}
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
