import { useState, type FormEvent } from "react";
import { Navigate, useNavigate, useSearchParams } from "react-router";
import { KeyRound } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { ApiError } from "../lib/apiClient";
import { CofactureLogo } from "../components/CofactureLogo";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Button } from "../components/ui/Button";
import { Banner } from "../components/ui/Banner";

// Página de destino del correo de "olvidé mi contraseña" (ver
// security/application.ForgotPasswordUseCase) -- mismo patrón que AcceptInvitePage: el token
// viaja en la URL (?token=...), nunca en el cuerpo de la página. A diferencia de esa, acá SÍ
// había una cuenta activa antes (con sesiones posiblemente abiertas en otros dispositivos), por
// eso ResetPasswordUseCase revoca todo lo demás y solo la sesión que se abre desde aquí queda viva.
export function ResetPasswordPage() {
  const { resetPassword, isAuthenticated, isReady } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  if (isReady && isAuthenticated) return <Navigate to="/" replace />;

  const passwordMismatch = confirmPassword.length > 0 && password !== confirmPassword;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (!token) {
      setError("El enlace no es válido — falta el token.");
      return;
    }
    if (password !== confirmPassword) {
      setError("Las contraseñas no coinciden");
      return;
    }
    setLoading(true);
    try {
      await resetPassword(token, password);
      navigate("/", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo restablecer la contraseña — el enlace puede haber vencido");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-(--bg-primary) px-4">
      <Card className="w-full max-w-sm">
        <div className="flex items-center gap-2 border-b border-(--border-light) px-4 py-3">
          <CofactureLogo className="h-5 w-5 text-(--accent-primary)" />
          <h1 className="text-base font-normal text-(--text-primary)">cofacture</h1>
        </div>
        <form className="flex flex-col gap-3 p-4" onSubmit={handleSubmit}>
          <p className="text-xs text-(--text-secondary)">Elige una nueva contraseña para tu cuenta.</p>
          {error && <Banner tone="danger">{error}</Banner>}
          {!token && <Banner tone="danger">Este enlace no tiene un token válido — pide un nuevo enlace de recuperación.</Banner>}
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
            id="confirmPassword"
            type="password"
            label="Confirmar contraseña"
            autoComplete="new-password"
            required
            minLength={8}
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            error={passwordMismatch ? "Las contraseñas no coinciden" : undefined}
          />
          <Button type="submit" icon={<KeyRound className="h-3.5 w-3.5" />} loading={loading} disabled={!token} className="w-full">
            Restablecer contraseña
          </Button>
        </form>
      </Card>
    </div>
  );
}
