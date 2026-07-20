import { useEffect, useState, type FormEvent } from "react";
import { Palette } from "lucide-react";
import { useToast } from "../../context/ToastContext";
import { ApiError } from "../../lib/apiClient";
import { getMySettings, updateBrandColor } from "../../lib/settings";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";

const DEFAULT_COLOR = "#14345C";

export function BrandColorForm() {
  const toast = useToast();
  const [color, setColor] = useState(DEFAULT_COLOR);
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(true);

  useEffect(() => {
    getMySettings()
      .then((s) => setColor(s.brand_color))
      .catch(() => {})
      .finally(() => setFetching(false));
  }, []);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      const updated = await updateBrandColor(color);
      setColor(updated.brand_color);
      toast.success("Color de marca guardado.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar el color");
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card className="flex flex-col gap-3 p-4">
      <h2 className="text-xs font-semibold text-(--text-primary)">Color de marca</h2>
      <p className="text-xs text-(--text-secondary)">
        Color principal que aparece en la cabecera de tus facturas PDF. El valor por defecto es el azul de cofacture.
      </p>
      <form className="flex items-end gap-3" onSubmit={handleSubmit}>
        <label className="flex flex-col gap-1">
          <span className="text-xs font-medium text-(--text-secondary)">Color (#RRGGBB)</span>
          <div className="flex items-center gap-2">
            <input
              type="color"
              value={color}
              disabled={fetching}
              onChange={(e) => setColor(e.target.value)}
              className="h-9 w-12 cursor-pointer rounded border border-(--border-color) bg-(--bg-primary) p-0.5"
            />
            <input
              type="text"
              value={color}
              disabled={fetching}
              onChange={(e) => {
                const v = e.target.value;
                if (/^#[0-9A-Fa-f]{0,6}$/.test(v)) setColor(v);
              }}
              maxLength={7}
              className="w-24 rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-xs font-mono text-(--text-primary) focus:outline-none focus:ring-1 focus:ring-(--accent-primary)"
            />
            <span
              className="h-7 w-7 rounded border border-(--border-color)"
              style={{ backgroundColor: /^#[0-9A-Fa-f]{6}$/.test(color) ? color : undefined }}
            />
          </div>
        </label>
        <Button
          type="submit"
          loading={loading}
          disabled={fetching || !/^#[0-9A-Fa-f]{6}$/.test(color)}
          icon={<Palette className="h-3.5 w-3.5" />}
        >
          Guardar
        </Button>
        {color !== DEFAULT_COLOR && (
          <Button
            type="button"
            variant="secondary"
            disabled={fetching || loading}
            onClick={() => setColor(DEFAULT_COLOR)}
          >
            Restablecer
          </Button>
        )}
      </form>
    </Card>
  );
}
