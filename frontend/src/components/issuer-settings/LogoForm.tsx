import { useEffect, useState, type FormEvent } from "react";
import { Image as ImageIcon, Trash2 } from "lucide-react";
import { useAuth } from "../../context/AuthContext";
import { useConfirm } from "../../context/ConfirmContext";
import { useToast } from "../../context/ToastContext";
import { useCanManage } from "../../hooks/useCanManage";
import { apiClient, ApiError } from "../../lib/apiClient";
import { fileToBase64 } from "../../lib/fileToBase64";
import type { UpdateCompanyPayload } from "../../lib/types";
import { AsyncImage } from "../ui/AsyncImage";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";
import { ManageOnlyHint } from "../ui/ManageOnlyHint";
import { StatusBadge } from "./SoftwareCertificateForm";

const CONTENT_TYPE_BY_MIME: Record<string, string> = {
  "image/png": "image/png",
  "image/jpeg": "image/jpeg",
  "image/webp": "image/webp",
};

// Logo para la representación gráfica en PDF — opcional, no es un secreto (a diferencia de
// SoftwareCertificateForm). Se sirve aparte (GET /companies/me/logo) para no inflar
// la respuesta de empresa con bytes de imagen en cada GET/PUT.
export function LogoForm() {
  const { activeCompany, updateCompany, deleteCompanyLogo } = useAuth();
  const canManage = useCanManage();
  const confirm = useConfirm();
  const toast = useToast();
  const [file, setFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [loadingPreview, setLoadingPreview] = useState(false);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (!activeCompany?.has_logo) {
      setPreviewUrl(null);
      return;
    }
    let objectUrl: string | null = null;
    setLoadingPreview(true);
    apiClient
      .getBlob("/companies/active/logo")
      .then((blob) => {
        objectUrl = URL.createObjectURL(blob);
        setPreviewUrl(objectUrl);
      })
      .catch(() => setPreviewUrl(null))
      .finally(() => setLoadingPreview(false));
    return () => {
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [activeCompany?.has_logo]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();

    if (!file) {
      toast.error("Selecciona una imagen primero.");
      return;
    }
    const contentType = CONTENT_TYPE_BY_MIME[file.type];
    if (!contentType) {
      toast.error("El logo debe ser una imagen PNG, JPG o WebP.");
      return;
    }

    const payload: UpdateCompanyPayload = {
      logo_base64: await fileToBase64(file),
      logo_content_type: contentType,
    };

    setLoading(true);
    try {
      await updateCompany(payload);
      toast.success("Logo guardado correctamente.");
      setFile(null);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar el logo");
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete() {
    if (!(await confirm("¿Eliminar el logo de la empresa?", { tone: "danger" }))) return;
    setDeleting(true);
    try {
      await deleteCompanyLogo();
      toast.success("Logo eliminado.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el logo");
    } finally {
      setDeleting(false);
    }
  }

  return (
    <Card className="flex flex-col gap-3 p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xs font-semibold text-(--text-primary)">Logo de la empresa</h2>
        <StatusBadge label="Logo" ok={activeCompany?.has_logo ?? false} />
      </div>
      <p className="text-xs text-(--text-secondary)">Aparece en la esquina superior izquierda de la representación gráfica (PDF) de tus facturas.</p>
      {canManage ? (
        <form className="flex items-end gap-3" onSubmit={handleSubmit}>
          {(loadingPreview || previewUrl) && (
            <AsyncImage src={previewUrl} loading={loadingPreview} alt="Logo actual" className="h-12 w-12" />
          )}
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-(--text-secondary)">Imagen (PNG, JPG o WebP)</span>
            <input
              type="file"
              accept="image/png,image/jpeg,image/webp"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              className="rounded border border-(--border-color) bg-(--bg-primary) px-3 py-1.5 text-xs text-(--text-primary)"
            />
          </label>
          <Button type="submit" loading={loading} icon={<ImageIcon className="h-3.5 w-3.5" />}>
            Guardar
          </Button>
          {activeCompany?.has_logo && (
            <Button type="button" variant="danger" loading={deleting} icon={<Trash2 className="h-3.5 w-3.5" />} onClick={handleDelete}>
              Eliminar logo
            </Button>
          )}
        </form>
      ) : (
        <div className="flex items-center gap-3">
          {(loadingPreview || previewUrl) && (
            <AsyncImage src={previewUrl} loading={loadingPreview} alt="Logo actual" className="h-12 w-12" />
          )}
          <ManageOnlyHint>{activeCompany?.has_logo ? "Solo un administrador o dueño puede cambiar el logo." : "Sin logo — solo un administrador o dueño puede subir uno."}</ManageOnlyHint>
        </div>
      )}
    </Card>
  );
}
