import { useState, type FormEvent } from "react";
import { ShieldCheck } from "lucide-react";
import { useAuth } from "../../context/AuthContext";
import { ApiError } from "../../lib/apiClient";
import { fileToBase64 } from "../../lib/fileToBase64";
import type { UpdateIssuerPayload } from "../../lib/types";
import { Input } from "../ui/Input";
import { Button } from "../ui/Button";
import { Banner } from "../ui/Banner";

function StatusBadge({ label, ok }: { label: string; ok: boolean }) {
  return (
    <span className={`flex items-center gap-1 text-xs ${ok ? "text-(--color-success)" : "text-(--text-muted)"}`}>
      {ok ? "✓" : "—"} {label}
    </span>
  );
}

// Completa software_id/software_pin/certificate/certificate_password DESPUÉS de creada la
// empresa (PUT /issuers/me) — la DIAN nunca devuelve estos secretos de vuelta, solo si están
// configurados o no (activeIssuer.has_software_credentials/has_certificate).
export function SoftwareCertificateForm() {
  const { activeIssuer, updateIssuer } = useAuth();
  const [softwareId, setSoftwareId] = useState("");
  const [softwarePin, setSoftwarePin] = useState("");
  const [certificateFile, setCertificateFile] = useState<File | null>(null);
  const [certificatePassword, setCertificatePassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    if ((certificateFile && !certificatePassword.trim()) || (!certificateFile && certificatePassword.trim())) {
      setError("El certificado y su contraseña se deben actualizar juntos.");
      return;
    }

    const payload: UpdateIssuerPayload = {};
    if (softwareId.trim()) payload.software_id = softwareId.trim();
    if (softwarePin.trim()) payload.software_pin = softwarePin.trim();
    if (certificateFile && certificatePassword.trim()) {
      payload.certificate_base64 = await fileToBase64(certificateFile);
      payload.certificate_password = certificatePassword.trim();
    }

    if (Object.keys(payload).length === 0) {
      setError("No hay ningún campo nuevo para guardar.");
      return;
    }

    setLoading(true);
    try {
      await updateIssuer(payload);
      setSuccess("Guardado correctamente.");
      setSoftwareId("");
      setSoftwarePin("");
      setCertificateFile(null);
      setCertificatePassword("");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo guardar la configuración");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded border border-(--border-color) p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xs font-semibold text-(--text-primary)">Software y certificado</h2>
        <div className="flex gap-3">
          <StatusBadge label="Software" ok={activeIssuer?.has_software_credentials ?? false} />
          <StatusBadge label="Certificado" ok={activeIssuer?.has_certificate ?? false} />
        </div>
      </div>
      {error && <Banner tone="danger">{error}</Banner>}
      {success && <Banner tone="success">{success}</Banner>}
      <form className="flex flex-col gap-3" onSubmit={handleSubmit}>
        {/* Grilla de 12 columnas fijas, ver company-form/IdentificationStep.tsx para el porqué. */}
        <div className="grid grid-cols-12 gap-3">
          <div className="col-span-3">
            <Input
              label="Software ID"
              value={softwareId}
              onChange={(e) => setSoftwareId(e.target.value)}
              placeholder={activeIssuer?.has_software_credentials ? "Ya configurado — deja vacío para no cambiar" : undefined}
            />
          </div>
          <div className="col-span-3">
            <Input
              label="Software PIN"
              type="password"
              value={softwarePin}
              onChange={(e) => setSoftwarePin(e.target.value)}
              placeholder={activeIssuer?.has_software_credentials ? "Ya configurado — deja vacío para no cambiar" : undefined}
            />
          </div>
          <label className="col-span-3 flex flex-col gap-1">
            <span className="text-xs font-medium text-(--text-secondary)">Certificado (.p12)</span>
            <input
              type="file"
              accept=".p12,.pfx"
              onChange={(e) => setCertificateFile(e.target.files?.[0] ?? null)}
              className="rounded border border-(--border-color) bg-(--bg-primary) px-3 py-1.5 text-xs text-(--text-primary)"
            />
          </label>
          <div className="col-span-3">
            <Input
              label="Contraseña del certificado"
              type="password"
              value={certificatePassword}
              onChange={(e) => setCertificatePassword(e.target.value)}
              placeholder={activeIssuer?.has_certificate ? "Ya configurado — deja vacío para no cambiar" : undefined}
            />
          </div>
        </div>
        <Button type="submit" loading={loading} icon={<ShieldCheck className="h-3.5 w-3.5" />} className="self-start">
          Guardar
        </Button>
      </form>
    </div>
  );
}
