import { useState } from "react";
import { AlertTriangle } from "lucide-react";
import { useAuth } from "../context/AuthContext";

// Aviso fijo arriba de todo, mostrado solo cuando AuthContext no logró confirmar la sesión
// contra el backend (servidor apagado, sin red) — distinto de un token inválido, que cierra
// sesión de verdad y manda a /login. Montado una sola vez en App.tsx, fuera de las rutas, para
// que cubra login/register/onboarding/dashboard por igual.
export function ConnectionBanner() {
  const { connectionError, retryConnection } = useAuth();
  const [retrying, setRetrying] = useState(false);

  if (!connectionError) return null;

  async function handleRetry() {
    setRetrying(true);
    try {
      await retryConnection();
    } finally {
      setRetrying(false);
    }
  }

  return (
    <div className="flex items-center justify-center gap-3 bg-(--color-danger-bg) px-3 py-1.5 text-xs text-(--color-danger-text)">
      <AlertTriangle className="h-3.5 w-3.5" />
      <span>No se pudo conectar con el servidor — tu sesión sigue activa, pero ningún dato se está cargando.</span>
      <button
        type="button"
        onClick={handleRetry}
        disabled={retrying}
        className="font-medium underline hover:no-underline disabled:cursor-not-allowed disabled:opacity-60"
      >
        {retrying ? "Reintentando…" : "Reintentar"}
      </button>
    </div>
  );
}
