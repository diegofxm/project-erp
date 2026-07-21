import { useState, type FormEvent } from "react";
import { useSearchParams, useNavigate } from "react-router";
import { Layers, KeyRound } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { ApiError } from "../lib/apiClient";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Button } from "../components/ui/Button";
import { Banner } from "../components/ui/Banner";

export function SetupPasswordPage() {
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";
  const { acceptInvite } = useAuth();
  const navigate = useNavigate();

  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  if (!token) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-(--bg-primary) px-4">
        <Card className="w-full max-w-sm p-6 text-center text-sm text-(--text-secondary)">
          Link inválido. Solicita al administrador que te envíe una nueva invitación.
        </Card>
      </div>
    );
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (password.length < 8) {
      setError("La contraseña debe tener al menos 8 caracteres");
      return;
    }
    if (password !== confirm) {
      setError("Las contraseñas no coinciden");
      return;
    }
    setLoading(true);
    try {
      await acceptInvite(token, password);
      navigate("/", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo configurar la contraseña");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-(--bg-primary) px-4">
      <Card className="w-full max-w-sm">
        <div className="flex items-center gap-2 border-b border-(--border-light) px-4 py-3">
          <Layers className="h-5 w-5 text-(--accent-primary)" />
          <h1 className="text-sm font-semibold text-(--text-primary)">apidian</h1>
        </div>
        <form className="flex flex-col gap-3 p-4" onSubmit={handleSubmit}>
          <div className="mb-1">
            <p className="text-sm font-medium text-(--text-primary)">Configura tu contraseña</p>
            <p className="mt-0.5 text-xs text-(--text-secondary)">
              Elige una contraseña segura para acceder a tu cuenta.
            </p>
          </div>
          {error && <Banner tone="danger">{error}</Banner>}
          <Input
            id="password"
            type="password"
            label="Nueva contraseña"
            autoComplete="new-password"
            required
            minLength={8}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Input
            id="confirm"
            type="password"
            label="Confirmar contraseña"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
          <Button type="submit" icon={<KeyRound className="h-3.5 w-3.5" />} loading={loading} className="w-full">
            Activar cuenta
          </Button>
        </form>
      </Card>
    </div>
  );
}
