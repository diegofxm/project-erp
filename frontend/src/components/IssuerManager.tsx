import { useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { ApiError } from "../lib/apiClient";
import type { CreateIssuerPayload, Issuer } from "../lib/types";
import { Button } from "./ui/Button";
import { Banner } from "./ui/Banner";
import { Spinner } from "./ui/Spinner";
import { CompanyForm } from "./company-form/CompanyForm";

// Lista las empresas a las que el usuario tiene acceso, deja cambiar de activa y crear una
// nueva (reusando CompanyForm tal cual) — sin chrome de página propio, para que tanto
// OnboardingPage (gate sin empresa activa) como IssuersPage (ya dentro del dashboard) lo
// envuelvan con su propio header/Card.
//
// Regla de ancho (ver docs/frontend-architecture.md): todo a todo el ancho disponible —
// listados y formularios por igual. CompanyForm redistribuye sus propios campos en una grilla
// auto-fit, no necesita que IssuerManager lo acote.
export function IssuerManager() {
  const { listIssuers, selectIssuer, createIssuer, activeIssuer } = useAuth();
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
      setShowForm(false);
      // listIssuers() de nuevo: la empresa recién creada no está en el `issuers` ya cargado
      // (a diferencia de handleSelect, que solo cambia cuál está activa entre las mismas).
      setIssuers(await listIssuers());
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo crear la empresa");
    } finally {
      setLoading(false);
    }
  }

  const hasIssuers = issuers !== null && issuers.length > 0;

  return (
    <div className="flex flex-col gap-3">
      {error && <Banner tone="danger">{error}</Banner>}

      {issuers === null && (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      )}

      {hasIssuers && !showForm && (
        <div className="flex flex-col gap-2">
          <p className="text-xs text-(--text-secondary)">Elige con cuál empresa quieres trabajar:</p>
          {issuers.map((issuer) => {
            const isActive = issuer.id === activeIssuer?.id;
            return (
              <button
                key={issuer.id}
                type="button"
                disabled={loading || isActive}
                onClick={() => handleSelect(issuer.id)}
                className="flex items-center justify-between rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 text-left text-xs hover:bg-(--bg-hover) disabled:cursor-default disabled:opacity-60"
              >
                <span className="font-medium text-(--text-primary)">{issuer.business_name}</span>
                <span className="flex items-center gap-2">
                  <span className="text-(--text-muted)">NIT {issuer.nit}</span>
                  {isActive && <span className="rounded bg-(--bg-selected) px-1.5 py-0.5 text-(--accent-primary)">Activa</span>}
                </span>
              </button>
            );
          })}
          <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowForm(true)}>
            Crear otra empresa
          </Button>
        </div>
      )}

      {(!hasIssuers || showForm) && issuers !== null && (
        <>
          {!hasIssuers && <p className="mb-3 text-xs text-(--text-secondary)">Todavía no tienes ninguna empresa — crea la primera.</p>}
          <CompanyForm onSubmit={handleCreate} loading={loading} onCancel={hasIssuers ? () => setShowForm(false) : undefined} />
        </>
      )}
    </div>
  );
}
