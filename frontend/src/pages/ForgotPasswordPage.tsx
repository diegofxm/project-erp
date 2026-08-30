import { useState, type FormEvent } from "react";
import { Link, Navigate } from "react-router";
import { Mail } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { ApiError } from "../lib/apiClient";
import { CofactureLogo } from "../components/CofactureLogo";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Button } from "../components/ui/Button";
import { Banner } from "../components/ui/Banner";

// Pide el correo y dispara ForgotPasswordUseCase (POST /auth/forgot-password) -- el backend
// SIEMPRE responde con el mismo mensaje exista o no la cuenta (no revela nada), así que esta
// página tampoco distingue: solo muestra éxito genérico o un error real de red/servidor.
export function ForgotPasswordPage() {
  const { forgotPassword, isAuthenticated, isReady } = useAuth();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);

  if (isReady && isAuthenticated) return <Navigate to="/" replace />;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await forgotPassword(email);
      setSent(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo procesar la solicitud — intenta de nuevo en unos minutos");
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
        {sent ? (
          <div className="flex flex-col gap-3 p-4">
            <Banner tone="success">
              Si el correo está registrado, te enviamos un enlace para restablecer tu contraseña — revisa tu bandeja (y spam).
            </Banner>
            <Link to="/login" className="text-center text-xs text-(--accent-primary) hover:underline">
              Volver a iniciar sesión
            </Link>
          </div>
        ) : (
          <form className="flex flex-col gap-3 p-4" onSubmit={handleSubmit}>
            <p className="text-xs text-(--text-secondary)">
              Escribe el correo de tu cuenta y te enviamos un enlace para restablecer la contraseña.
            </p>
            {error && <Banner tone="danger">{error}</Banner>}
            <Input
              id="email"
              type="email"
              label="Correo"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <Button type="submit" icon={<Mail className="h-3.5 w-3.5" />} loading={loading} className="w-full">
              Enviar enlace de recuperación
            </Button>
            <Link to="/login" className="text-center text-xs text-(--text-secondary) hover:text-(--accent-primary) hover:underline">
              Volver a iniciar sesión
            </Link>
          </form>
        )}
      </Card>
    </div>
  );
}
