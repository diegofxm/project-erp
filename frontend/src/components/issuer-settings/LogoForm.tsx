import { useEffect, useState, type FormEvent } from "react";
import { Image as ImageIcon, Trash2 } from "lucide-react";
import { useAuth } from "../../context/AuthContext";
import { apiClient, ApiError } from "../../lib/apiClient";
import { fileToBase64 } from "../../lib/fileToBase64";
import type { UpdateIssuerPayload } from "../../lib/types";
import { Button } from "../ui/Button";
import { Banner } from "../ui/Banner";
import { StatusBadge } from "./SoftwareCertificateForm";

// apidian solo acepta estos dos formatos (mismos que internal/pdf necesita para incrustar la
// imagen, ver issuers.ErrInvalidLogoContentType en docs/apidian-architecture.md sección 9.39).
const CONTENT_TYPE_BY_MIME: Record<string, string> = {
  "image/png": "png",
  "image/jpeg": "jpeg",
};

// Logo para la representación gráfica en PDF — opcional, no es un secreto (a diferencia de
// SoftwareCertificateForm). Se sirve aparte (GET /issuers/me/logo) para no inflar
// issuerResponse con bytes de imagen en cada GET/PUT.
export function LogoForm() {
  const { activeIssuer, updateIssuer, deleteIssuerLogo } = useAuth();
  const [file, setFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (!activeIssuer?.has_logo) {
      setPreviewUrl(null);
      return;
    }
    let objectUrl: string | null = null;
    apiClient
      .getBlob("/issuers/me/logo")
      .then((blob) => {
        objectUrl = URL.createObjectURL(blob);
        setPreviewUrl(objectUrl);
      })
      .catch(() => setPreviewUrl(null));
    return () => {
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [activeIssuer?.has_logo]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    if (!file) {
      setError("Selecciona una imagen primero.");
      return;
    }
    const contentType = CONTENT_TYPE_BY_MIME[file.type];
    if (!contentType) {
      setError("El logo debe ser una imagen PNG o JPG.");
      return;
    }

    const payload: UpdateIssuerPayload = {
      logo_base64: await fileToBase64(file),
      logo_content_type: contentType,
    };

    setLoading(true);
    try {
      await updateIssuer(payload);
      setSuccess("Logo guardado correctamente.");
      setFile(null);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar el logo");
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete() {
    if (!window.confirm("¿Eliminar el logo de la empresa?")) return;
    setError(null);
    setSuccess(null);
    setDeleting(true);
    try {
      await deleteIssuerLogo();
      setSuccess("Logo eliminado.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo eliminar el logo");
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded border border-(--border-color) p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xs font-semibold text-(--text-primary)">Logo de la empresa</h2>
        <StatusBadge label="Logo" ok={activeIssuer?.has_logo ?? false} />
      </div>
      <p className="text-xs text-(--text-secondary)">Aparece en la esquina superior izquierda de la representación gráfica (PDF) de tus facturas.</p>
      {error && <Banner tone="danger">{error}</Banner>}
      {success && <Banner tone="success">{success}</Banner>}
      <form className="flex items-end gap-3" onSubmit={handleSubmit}>
        {previewUrl && (
          <img src={previewUrl} alt="Logo actual" className="h-12 w-12 rounded border border-(--border-color) object-contain" />
        )}
        <label className="flex flex-col gap-1">
          <span className="text-xs font-medium text-(--text-secondary)">Imagen (PNG o JPG)</span>
          <input
            type="file"
            accept="image/png,image/jpeg"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            className="rounded border border-(--border-color) bg-(--bg-primary) px-3 py-1.5 text-xs text-(--text-primary)"
          />
        </label>
        <Button type="submit" loading={loading} icon={<ImageIcon className="h-3.5 w-3.5" />}>
          Guardar
        </Button>
        {activeIssuer?.has_logo && (
          <Button type="button" variant="danger" loading={deleting} icon={<Trash2 className="h-3.5 w-3.5" />} onClick={handleDelete}>
            Eliminar logo
          </Button>
        )}
      </form>
    </div>
  );
}
