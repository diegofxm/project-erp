import { useEffect, useState, type FormEvent } from "react";
import { useParams } from "react-router";
import { CheckCircle2, UserPlus } from "lucide-react";
import { ApiError } from "../lib/apiClient";
import { getPublicIssuer, registerPublicCustomer } from "../lib/publicRegistration";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Select } from "../components/ui/Select";
import { Spinner } from "../components/ui/Spinner";

// Lista corta a mano (no el catálogo identification_types completo): esta página no tiene
// sesión, y los identification-types de apidian son una ruta autenticada (ver
// docs/apidian-architecture.md sección 9.41) — estos 5 cubren el caso real de mostrador.
const IDENTIFICATION_TYPES = [
  { code: "13", name: "Cédula de Ciudadanía" },
  { code: "31", name: "NIT" },
  { code: "22", name: "Cédula de Extranjería" },
  { code: "41", name: "Pasaporte" },
  { code: "91", name: "NUIP" },
];

// Página pública de autorregistro (patrón QR/D1, sección 9.41) — sin Navbar/Sidebar, sin
// sesión: el emisor imprime el link/QR a esta ruta en su mostrador, el cliente la escanea
// desde el celular y se autorregistra sin que nadie tenga que digitar nada por él.
export function PublicCustomerRegisterPage() {
  const { issuerId } = useParams<{ issuerId: string }>();
  const [businessName, setBusinessName] = useState<string | null>(null);
  const [loadingIssuer, setLoadingIssuer] = useState(true);
  const [notFound, setNotFound] = useState(false);

  const [typeCode, setTypeCode] = useState("13");
  const [number, setNumber] = useState("");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);

  useEffect(() => {
    if (!issuerId) return;
    getPublicIssuer(issuerId)
      .then((iss) => setBusinessName(iss.business_name))
      .catch(() => setNotFound(true))
      .finally(() => setLoadingIssuer(false));
  }, [issuerId]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!issuerId) return;
    setError(null);
    setSubmitting(true);
    try {
      await registerPublicCustomer(issuerId, {
        identification: { number, type_code: typeCode },
        name,
        email: email || undefined,
        phone: phone || undefined,
      });
      setDone(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo completar el registro");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-(--bg-primary) px-4">
      <Card className="w-full max-w-sm">
        {loadingIssuer ? (
          <div className="flex min-h-32 items-center justify-center p-4">
            <Spinner className="h-5 w-5 text-(--text-muted)" />
          </div>
        ) : notFound ? (
          <div className="p-4">
            <Banner tone="danger">Este enlace de registro no es válido.</Banner>
          </div>
        ) : done ? (
          <div className="flex flex-col items-center gap-2 p-6 text-center">
            <CheckCircle2 className="h-8 w-8 text-(--color-success-text)" />
            <p className="text-sm font-medium text-(--text-primary)">¡Listo!</p>
            <p className="text-xs text-(--text-secondary)">
              Quedaste registrado — {businessName} ya puede facturarte electrónicamente.
            </p>
          </div>
        ) : (
          <>
            <div className="flex items-center gap-2 border-b border-(--border-light) px-4 py-3">
              <UserPlus className="h-5 w-5 text-(--accent-primary)" />
              <h1 className="text-sm font-semibold text-(--text-primary)">Regístrate como cliente de {businessName}</h1>
            </div>
            <form className="flex flex-col gap-3 p-4" onSubmit={handleSubmit}>
              {error && <Banner tone="danger">{error}</Banner>}
              <Select label="Tipo de identificación" required value={typeCode} onChange={(e) => setTypeCode(e.target.value)}>
                {IDENTIFICATION_TYPES.map((t) => (
                  <option key={t.code} value={t.code}>
                    {t.name}
                  </option>
                ))}
              </Select>
              <Input label="Número de identificación" required value={number} onChange={(e) => setNumber(e.target.value)} />
              <Input label="Nombre completo" required value={name} onChange={(e) => setName(e.target.value)} />
              <Input label="Correo (opcional)" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
              <Input label="Teléfono (opcional)" value={phone} onChange={(e) => setPhone(e.target.value)} />
              <Button type="submit" icon={<UserPlus className="h-3.5 w-3.5" />} loading={submitting} className="w-full">
                Registrarme
              </Button>
            </form>
          </>
        )}
      </Card>
    </div>
  );
}
