import { useEffect, useState, type FormEvent } from "react";
import { Building2, Plus } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { ApiError } from "../lib/apiClient";
import type { CreateIssuerPayload, Issuer } from "../lib/types";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Button } from "../components/ui/Button";
import { Banner } from "../components/ui/Banner";

const EMPTY_FORM: CreateIssuerPayload = {
  nit: "",
  check_digit: "",
  business_name: "",
  identification_type_code: "31",
  department_code: "",
  municipality_code: "",
  address_line: "",
  email: "",
  environment: "2",
};

// Gate previo al dashboard cuando el usuario no tiene empresa activa (0 o 2+ vinculadas, ver
// middleware.RequireTenant en apidian) — lista las que ya tiene acceso y/o deja crear una
// nueva. Solo pide los campos mínimos que el backend exige (nit/business_name/environment);
// software/PIN/certificado se completan después en una fase de configuración aparte.
export function OnboardingPage() {
  const { listIssuers, selectIssuer, createIssuer, logout } = useAuth();
  const [issuers, setIssuers] = useState<Issuer[] | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<CreateIssuerPayload>(EMPTY_FORM);
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

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await createIssuer(form);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo crear la empresa");
    } finally {
      setLoading(false);
    }
  }

  function setField<K extends keyof CreateIssuerPayload>(key: K, value: CreateIssuerPayload[K]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  const hasIssuers = issuers !== null && issuers.length > 0;

  return (
    <div className="flex min-h-screen items-center justify-center bg-(--bg-primary) px-4 py-8">
      <Card className="w-full max-w-md">
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
            <form className="flex flex-col gap-3" onSubmit={handleCreate}>
              {!hasIssuers && <p className="text-xs text-(--text-secondary)">Todavía no tienes ninguna empresa — crea la primera.</p>}
              <div className="grid grid-cols-2 gap-2">
                <Input label="NIT" required value={form.nit} onChange={(e) => setField("nit", e.target.value)} />
                <Input label="Dígito verificación" required value={form.check_digit} onChange={(e) => setField("check_digit", e.target.value)} />
              </div>
              <Input label="Razón social" required value={form.business_name} onChange={(e) => setField("business_name", e.target.value)} />
              <Input
                label="Correo de la empresa"
                type="email"
                required
                value={form.email}
                onChange={(e) => setField("email", e.target.value)}
              />
              <Input label="Dirección" required value={form.address_line} onChange={(e) => setField("address_line", e.target.value)} />
              <div className="grid grid-cols-2 gap-2">
                <Input
                  label="Código departamento"
                  required
                  value={form.department_code}
                  onChange={(e) => setField("department_code", e.target.value)}
                />
                <Input
                  label="Código municipio"
                  required
                  value={form.municipality_code}
                  onChange={(e) => setField("municipality_code", e.target.value)}
                />
              </div>
              <label className="flex flex-col gap-1">
                <span className="text-xs font-medium text-(--text-secondary)">Ambiente</span>
                <select
                  className="w-full rounded border border-(--border-color) bg-(--bg-primary) px-3 py-1.5 text-xs text-(--text-primary)"
                  value={form.environment}
                  onChange={(e) => setField("environment", e.target.value as CreateIssuerPayload["environment"])}
                >
                  <option value="2">Habilitación</option>
                  <option value="1">Producción</option>
                </select>
              </label>
              <div className="flex gap-2">
                {hasIssuers && (
                  <Button type="button" variant="secondary" className="flex-1" onClick={() => setShowForm(false)}>
                    Cancelar
                  </Button>
                )}
                <Button type="submit" loading={loading} className="flex-1">
                  Crear empresa
                </Button>
              </div>
            </form>
          )}
        </div>
      </Card>
    </div>
  );
}
