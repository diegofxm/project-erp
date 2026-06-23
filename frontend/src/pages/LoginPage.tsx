import { useState, type FormEvent } from "react";
import { Link, Navigate, useNavigate } from "react-router";
import { Layers, LogIn } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { ApiError } from "../lib/apiClient";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Button } from "../components/ui/Button";
import { Banner } from "../components/ui/Banner";

export function LoginPage() {
  const { login, isAuthenticated, isReady } = useAuth();
  const navigate = useNavigate();
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
      await login({ email, password });
      navigate("/", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo iniciar sesión");
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
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Button type="submit" icon={<LogIn className="h-3.5 w-3.5" />} loading={loading} className="w-full">
            Iniciar sesión
          </Button>
          <p className="text-center text-xs text-(--text-secondary)">
            ¿No tienes cuenta?{" "}
            <Link to="/register" className="font-medium text-(--accent-primary) hover:text-(--accent-hover)">
              Regístrate
            </Link>
          </p>
        </form>
      </Card>
    </div>
  );
}
