import { useEffect, useState } from "react";
import { Building2, Plus } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { ApiError } from "../lib/apiClient";
import type { CreateIssuerPayload, Issuer } from "../lib/types";
import { Card } from "../components/ui/Card";
import { Button } from "../components/ui/Button";
import { Banner } from "../components/ui/Banner";
import { CompanyForm } from "../components/company-form/CompanyForm";

// Gate previo al dashboard cuando el usuario no tiene empresa activa (0 o 2+ vinculadas, ver
// middleware.RequireTenant en apidian) — lista las que ya tiene acceso y/o deja crear una
// nueva. El formulario de creación (CompanyForm) pide todo lo que la DIAN exige del emisor
// salvo la configuración técnica (software/PIN/certificado), que se completa después en una
// fase de configuración aparte.
export function OnboardingPage() {
  const { listIssuers, selectIssuer, createIssuer, logout } = useAuth();
  const [issuers, setIssuers] = useState<Issuer[] | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    listIssuers()
      .then(setIssuers)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar tus empresas"));
  }, [listIssuers]);

  async function handleSelect(id: string) {
    setError(null);
    setLoading(true);
    try {
      await selectIssuer(id);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo seleccionar la empresa");
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate(payload: CreateIssuerPayload) {
    setError(null);
    setLoading(true);
    try {
      await createIssuer(payload);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo crear la empresa");
    } finally {
      setLoading(false);
    }
  }

  const hasIssuers = issuers !== null && issuers.length > 0;

  return (
    <div className="flex min-h-screen items-center justify-center bg-(--bg-primary) px-4 py-8">
      <Card className="w-full max-w-2xl">
        <div className="flex items-center justify-between border-b border-(--border-light) px-4 py-3">
          <div className="flex items-center gap-2">
            <Building2 className="h-5 w-5 text-(--accent-primary)" />
            <h1 className="text-sm font-semibold text-(--text-primary)">Tu empresa</h1>
          </div>
          <button type="button" onClick={logout} className="text-xs text-(--text-secondary) hover:text-(--text-primary)">
            Cerrar sesión
          </button>
        </div>

        <div className="flex flex-col gap-3 p-4">
          {error && <Banner tone="danger">{error}</Banner>}

          {hasIssuers && !showForm && (
            <div className="flex flex-col gap-2">
              <p className="text-xs text-(--text-secondary)">Elige con cuál empresa quieres trabajar:</p>
              {issuers.map((issuer) => (
                <button
                  key={issuer.id}
                  type="button"
                  disabled={loading}
                  onClick={() => handleSelect(issuer.id)}
                  className="flex items-center justify-between rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 text-left text-xs hover:bg-(--bg-hover) disabled:opacity-60"
                >
                  <span className="font-medium text-(--text-primary)">{issuer.business_name}</span>
                  <span className="text-(--text-muted)">NIT {issuer.nit}</span>
                </button>
              ))}
              <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowForm(true)}>
                Crear otra empresa
              </Button>
            </div>
          )}

          {(!hasIssuers || showForm) && issuers !== null && (
            <>
              {!hasIssuers && <p className="text-xs text-(--text-secondary)">Todavía no tienes ninguna empresa — crea la primera.</p>}
              <CompanyForm onSubmit={handleCreate} loading={loading} onCancel={hasIssuers ? () => setShowForm(false) : undefined} />
            </>
          )}
        </div>
      </Card>
    </div>
  );
}
