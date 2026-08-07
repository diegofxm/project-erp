import { useState, type FormEvent } from "react";
import { ShieldCheck, Trash2, KeyRound, BadgeCheck, ClipboardList } from "lucide-react";
import { useAuth } from "../../context/AuthContext";
import { useConfirm } from "../../context/ConfirmContext";
import { useToast } from "../../context/ToastContext";
import { useCanManage } from "../../hooks/useCanManage";
import { ApiError } from "../../lib/apiClient";
import { fileToBase64 } from "../../lib/fileToBase64";
import type { UpdateCompanyPayload } from "../../lib/types";
import { Card } from "../ui/Card";
import { Input } from "../ui/Input";
import { Button } from "../ui/Button";
import { ManageOnlyHint } from "../ui/ManageOnlyHint";

// Exportado: LogoForm.tsx reusa el mismo indicador "✓/—" para has_logo.
export function StatusBadge({ label, ok }: { label: string; ok: boolean }) {
  return (
    <span className={`flex items-center gap-1 text-xs ${ok ? "text-(--color-success)" : "text-(--text-muted)"}`}>
      {ok ? "✓" : "—"} {label}
    </span>
  );
}

// Semáforo de vencimiento del certificado. Verde > 30 días, amarillo ≤ 30 días, rojo vencido.
function CertStatusBadge({ expiresAt }: { expiresAt: string }) {
  const expiry = new Date(expiresAt);
  const now = new Date();
  const daysLeft = Math.floor((expiry.getTime() - now.getTime()) / 86_400_000);

  let label: string;
  let cls: string;
  if (daysLeft < 0) {
    label = "Vencido";
    cls = "bg-(--color-danger-bg) text-(--color-danger-text)";
  } else if (daysLeft <= 30) {
    label = `Alerta — vence en ${daysLeft} día${daysLeft === 1 ? "" : "s"}`;
    cls = "bg-(--color-warning-bg) text-(--color-warning-text)";
  } else {
    label = "Activo";
    cls = "bg-(--color-success-bg) text-(--color-success-text)";
  }

  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${cls}`}>
      <span className="h-1.5 w-1.5 rounded-full bg-current" />
      {label}
    </span>
  );
}

function SoftwareSection() {
  const { activeCompany, updateCompany, deleteCompanySoftware } = useAuth();
  const confirm = useConfirm();
  const toast = useToast();
  const [softwareId, setSoftwareId] = useState("");
  const [softwarePin, setSoftwarePin] = useState("");
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const configured = activeCompany?.has_software_credentials ?? false;

  async function handleSave(e: FormEvent) {
    e.preventDefault();
    if (!softwareId.trim() || !softwarePin.trim()) {
      toast.error("El Software ID y el PIN son obligatorios.");
      return;
    }
    const payload: UpdateCompanyPayload = { software_id: softwareId.trim(), software_pin: softwarePin.trim() };
    setLoading(true);
    try {
      await updateCompany(payload);
      toast.success("Software guardado correctamente.");
      setSoftwareId("");
      setSoftwarePin("");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar el software");
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete() {
    if (!(await confirm("¿Eliminar las credenciales de software? La empresa no podrá emitir documentos hasta volver a configurarlas.", { tone: "danger", title: "Eliminar software DIAN" }))) return;
    setDeleting(true);
    try {
      await deleteCompanySoftware();
      toast.success("Credenciales de software eliminadas.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el software");
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <KeyRound className="h-3.5 w-3.5 text-(--text-muted)" />
          <h3 className="text-xs font-semibold text-(--text-primary)">Software DIAN</h3>
        </div>
        {configured && (
          <Button type="button" variant="danger" loading={deleting} icon={<Trash2 className="h-3 w-3" />} onClick={handleDelete}>
            Eliminar
          </Button>
        )}
      </div>

      {configured ? (
        <div className="flex flex-col gap-1.5 rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2.5 text-xs">
          <div className="flex gap-2">
            <span className="w-24 shrink-0 text-(--text-muted)">Software ID</span>
            <span className="font-mono text-(--text-primary)">{activeCompany?.software_id ?? "—"}</span>
          </div>
          <div className="flex gap-2">
            <span className="w-24 shrink-0 text-(--text-muted)">PIN</span>
            <span className="text-(--text-secondary)">configurado</span>
          </div>
        </div>
      ) : (
        <form className="flex flex-col gap-3" onSubmit={handleSave}>
          <div className="grid grid-cols-2 gap-3">
            <Input label="Software ID" value={softwareId} onChange={(e) => setSoftwareId(e.target.value)} />
            <Input label="Software PIN" type="password" value={softwarePin} onChange={(e) => setSoftwarePin(e.target.value)} />
          </div>
          <Button type="submit" loading={loading} icon={<ShieldCheck className="h-3.5 w-3.5" />} className="self-start">
            Guardar software
          </Button>
        </form>
      )}
    </div>
  );
}

function CertificateSection() {
  const { activeCompany, updateCompany, deleteCompanyCertificate } = useAuth();
  const confirm = useConfirm();
  const toast = useToast();
  const [certFile, setCertFile] = useState<File | null>(null);
  const [certPassword, setCertPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const configured = activeCompany?.has_certificate ?? false;

  async function handleSave(e: FormEvent) {
    e.preventDefault();
    if (!certFile || !certPassword.trim()) {
      toast.error("El archivo .p12 y la contraseña son obligatorios.");
      return;
    }
    const payload: UpdateCompanyPayload = {
      certificate_base64: await fileToBase64(certFile),
      certificate_password: certPassword.trim(),
    };
    setLoading(true);
    try {
      await updateCompany(payload);
      toast.success("Certificado guardado correctamente.");
      setCertFile(null);
      setCertPassword("");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar el certificado");
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete() {
    if (!(await confirm("¿Eliminar el certificado digital? La empresa no podrá firmar ni enviar documentos a la DIAN hasta volver a configurarlo.", { tone: "danger", title: "Eliminar certificado" }))) return;
    setDeleting(true);
    try {
      await deleteCompanyCertificate();
      toast.success("Certificado eliminado.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el certificado");
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <BadgeCheck className="h-3.5 w-3.5 text-(--text-muted)" />
          <h3 className="text-xs font-semibold text-(--text-primary)">Certificado digital</h3>
        </div>
        {configured && (
          <Button type="button" variant="danger" loading={deleting} icon={<Trash2 className="h-3 w-3" />} onClick={handleDelete}>
            Eliminar
          </Button>
        )}
      </div>

      {configured ? (
        <div className="flex flex-col gap-1.5 rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2.5 text-xs">
          {activeCompany?.certificate_subject && (
            <div className="flex gap-2">
              <span className="w-24 shrink-0 text-(--text-muted)">Titular</span>
              <span className="text-(--text-primary)">{activeCompany.certificate_subject}</span>
            </div>
          )}
          {activeCompany?.certificate_issuer_cn && (
            <div className="flex gap-2">
              <span className="w-24 shrink-0 text-(--text-muted)">Emisor</span>
              <span className="text-(--text-secondary)">{activeCompany.certificate_issuer_cn}</span>
            </div>
          )}
          {activeCompany?.certificate_expires_at && (
            <div className="flex items-center gap-2">
              <span className="w-24 shrink-0 text-(--text-muted)">Vencimiento</span>
              <span className="text-(--text-secondary)">
                {new Date(activeCompany.certificate_expires_at).toLocaleDateString("es-CO", { day: "numeric", month: "short", year: "numeric" })}
              </span>
              <CertStatusBadge expiresAt={activeCompany.certificate_expires_at} />
            </div>
          )}
        </div>
      ) : (
        <form className="flex flex-col gap-3" onSubmit={handleSave}>
          <div className="grid grid-cols-2 gap-3">
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-(--text-secondary)">Archivo (.p12 / .pfx)</span>
              <input
                type="file"
                accept=".p12,.pfx"
                onChange={(e) => setCertFile(e.target.files?.[0] ?? null)}
                className="rounded border border-(--border-color) bg-(--bg-primary) px-3 py-1.5 text-xs text-(--text-primary)"
              />
            </label>
            <Input label="Contraseña del certificado" type="password" value={certPassword} onChange={(e) => setCertPassword(e.target.value)} />
          </div>
          <Button type="submit" loading={loading} icon={<ShieldCheck className="h-3.5 w-3.5" />} className="self-start">
            Guardar certificado
          </Button>
        </form>
      )}
    </div>
  );
}

function NeSoftwareSection() {
  const { activeCompany, updateCompany, deleteCompanyNeSoftware } = useAuth();
  const confirm = useConfirm();
  const toast = useToast();
  const [softwareId, setSoftwareId] = useState("");
  const [softwarePin, setSoftwarePin] = useState("");
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const configured = activeCompany?.has_ne_software_credentials ?? false;

  async function handleSave(e: FormEvent) {
    e.preventDefault();
    if (!softwareId.trim() || !softwarePin.trim()) {
      toast.error("El Software ID NE y el PIN son obligatorios.");
      return;
    }
    const payload: UpdateCompanyPayload = { ne_software_id: softwareId.trim(), ne_software_pin: softwarePin.trim() };
    setLoading(true);
    try {
      await updateCompany(payload);
      toast.success("Software NE guardado correctamente.");
      setSoftwareId("");
      setSoftwarePin("");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar el software NE");
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete() {
    if (!(await confirm("¿Eliminar las credenciales de software NE? No podrás emitir Nómina Electrónica hasta volver a configurarlas.", { tone: "danger", title: "Eliminar software NE" }))) return;
    setDeleting(true);
    try {
      await deleteCompanyNeSoftware();
      toast.success("Credenciales de software NE eliminadas.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el software NE");
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <ClipboardList className="h-3.5 w-3.5 text-(--text-muted)" />
          <h3 className="text-xs font-semibold text-(--text-primary)">Software NE (Nómina Electrónica)</h3>
        </div>
        {configured && (
          <Button type="button" variant="danger" loading={deleting} icon={<Trash2 className="h-3 w-3" />} onClick={handleDelete}>
            Eliminar
          </Button>
        )}
      </div>

      {configured ? (
        <div className="flex flex-col gap-1.5 rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2.5 text-xs">
          <div className="flex gap-2">
            <span className="w-24 shrink-0 text-(--text-muted)">Software ID</span>
            <span className="font-mono text-(--text-primary)">{activeCompany?.ne_software_id ?? "—"}</span>
          </div>
          <div className="flex gap-2">
            <span className="w-24 shrink-0 text-(--text-muted)">PIN</span>
            <span className="text-(--text-secondary)">configurado</span>
          </div>
        </div>
      ) : (
        <form className="flex flex-col gap-3" onSubmit={handleSave}>
          <div className="grid grid-cols-2 gap-3">
            <Input label="Software ID NE" value={softwareId} onChange={(e) => setSoftwareId(e.target.value)} />
            <Input label="Software PIN NE" type="password" value={softwarePin} onChange={(e) => setSoftwarePin(e.target.value)} />
          </div>
          <Button type="submit" loading={loading} icon={<ShieldCheck className="h-3.5 w-3.5" />} className="self-start">
            Guardar software NE
          </Button>
        </form>
      )}
    </div>
  );
}

// Completa software/PIN/certificado DESPUÉS de creada la empresa (PUT /companies/me).
// FE y DS comparten el mismo software; NE tiene credenciales propias.
export function SoftwareCertificateForm() {
  const canManage = useCanManage();
  return (
    <Card className="flex flex-col gap-5 p-4">
      <h2 className="text-xs font-semibold text-(--text-primary)">Software y certificado</h2>
      {!canManage && <ManageOnlyHint />}
      <fieldset disabled={!canManage} className="contents">
        <SoftwareSection />
        <div className="border-t border-(--border-light)" />
        <NeSoftwareSection />
        <div className="border-t border-(--border-light)" />
        <CertificateSection />
      </fieldset>
    </Card>
  );
}
