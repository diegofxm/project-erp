import { useState, type FormEvent } from "react";
import { Link, Navigate, useNavigate } from "react-router";
import { Layers, UserPlus } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { ApiError } from "../lib/apiClient";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Button } from "../components/ui/Button";
import { Banner } from "../components/ui/Banner";

// Registro desacoplado de la empresa (sección 9.32 de apidian-architecture.md): este form solo
// crea el usuario. Crear/seleccionar empresa pasa en OnboardingPage, después de loguearse.
export function RegisterPage() {
  const { register, isAuthenticated, isReady } = useAuth();
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  if (isReady && isAuthenticated) return <Navigate to="/" replace />;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await register({ name, email, password });
      navigate("/", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo completar el registro");
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
          {error && <Banner tone="danger">{error}</Banner>}
          <Input id="name" label="Nombre" autoComplete="name" required value={name} onChange={(e) => setName(e.target.value)} />
          <Input
            id="email"
            type="email"
            label="Correo"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <Input
            id="password"
            type="password"
            label="Contraseña"
            autoComplete="new-password"
            required
            minLength={8}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Button type="submit" icon={<UserPlus className="h-3.5 w-3.5" />} loading={loading} className="w-full">
            Crear cuenta
          </Button>
          <p className="text-center text-xs text-(--text-secondary)">
            ¿Ya tienes cuenta?{" "}
            <Link to="/login" className="font-medium text-(--accent-primary) hover:text-(--accent-hover)">
              Inicia sesión
            </Link>
          </p>
        </form>
      </Card>
    </div>
  );
}
