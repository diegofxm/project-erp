import { useEffect, useState } from "react";
import { KeyRound, User } from "lucide-react";
import { ApiError } from "../lib/apiClient";
import { useAuth } from "../context/AuthContext";
import { useToast } from "../context/ToastContext";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";

export function SettingsAccountPage() {
  const { user, updateProfile, changePassword } = useAuth();
  const toast = useToast();

  const [name, setName] = useState(user?.name ?? "");
  const [email, setEmail] = useState(user?.email ?? "");
  const [saving, setSaving] = useState(false);

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [changingPassword, setChangingPassword] = useState(false);

  useEffect(() => {
    setName(user?.name ?? "");
    setEmail(user?.email ?? "");
  }, [user?.name, user?.email]);

  const dirty = name !== (user?.name ?? "") || email !== (user?.email ?? "");
  const passwordMismatch = confirmPassword.length > 0 && newPassword !== confirmPassword;
  const canChangePassword = currentPassword.length > 0 && newPassword.length >= 8 && newPassword === confirmPassword;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!dirty) return;
    setSaving(true);
    try {
      await updateProfile(name, email);
      toast.success("Perfil actualizado.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo actualizar el perfil");
    } finally {
      setSaving(false);
    }
  }

  async function handleChangePassword(e: React.FormEvent) {
    e.preventDefault();
    if (!canChangePassword) return;
    setChangingPassword(true);
    try {
      await changePassword(currentPassword, newPassword);
      toast.success("Contraseña actualizada.");
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo cambiar la contraseña");
    } finally {
      setChangingPassword(false);
    }
  }

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Configuración", to: "/settings/general" }, { label: "Mi cuenta" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <User className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Mi cuenta
      </h1>
      <Card className="w-full max-w-sm p-4">
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <label className="text-xs text-(--text-secondary)">Nombre</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Tu nombre"
              required
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-(--text-secondary)">Correo electrónico</label>
            <Input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="correo@ejemplo.com"
              required
            />
          </div>
          <div className="flex justify-end">
            <Button type="submit" variant="primary" loading={saving} disabled={!dirty}>
              Guardar cambios
            </Button>
          </div>
        </form>
      </Card>

      <Card className="mt-4 w-full max-w-sm p-4">
        <h2 className="mb-3 flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-(--text-secondary)">
          <KeyRound className="h-3.5 w-3.5 shrink-0 text-(--accent-primary)" />
          Cambiar contraseña
        </h2>
        <form onSubmit={handleChangePassword} className="flex flex-col gap-3">
          <Input
            type="password"
            label="Contraseña actual"
            autoComplete="current-password"
            required
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
          />
          <Input
            type="password"
            label="Nueva contraseña"
            autoComplete="new-password"
            required
            minLength={8}
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
          />
          <Input
            type="password"
            label="Confirmar nueva contraseña"
            autoComplete="new-password"
            required
            minLength={8}
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            error={passwordMismatch ? "Las contraseñas no coinciden" : undefined}
          />
          <div className="flex justify-end">
            <Button type="submit" variant="primary" loading={changingPassword} disabled={!canChangePassword}>
              Cambiar contraseña
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
