import { useEffect, useState, type FormEvent } from "react";
import { useParams } from "react-router";
import { CheckCircle2, UserPlus, X } from "lucide-react";
import { ApiError } from "../lib/apiClient";
import { getPublicIssuer, getPublicIssuerLogoBlobUrl, registerPublicCustomer } from "../lib/publicRegistration";
import { AsyncImage } from "../components/ui/AsyncImage";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Select } from "../components/ui/Select";
import { Spinner } from "../components/ui/Spinner";

const IDENTIFICATION_TYPES = [
  { code: "13", name: "Cédula de Ciudadanía" },
  { code: "31", name: "NIT" },
  { code: "22", name: "Cédula de Extranjería" },
  { code: "41", name: "Pasaporte" },
  { code: "91", name: "NUIP" },
];

export function PublicCustomerRegisterPage() {
  const { issuerId } = useParams<{ issuerId: string }>();
  const [businessName, setBusinessName] = useState<string | null>(null);
  const [logoUrl, setLogoUrl] = useState<string | null>(null);
  const [loadingIssuer, setLoadingIssuer] = useState(true);
  const [loadingLogo, setLoadingLogo] = useState(false);
  const [notFound, setNotFound] = useState(false);

  const [typeCode, setTypeCode] = useState("13");
  const [number, setNumber] = useState("");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [done, setDone] = useState(false);

  useEffect(() => {
    if (!issuerId) return;
    let objectUrl: string | null = null;
    getPublicIssuer(issuerId)
      .then((iss) => {
        setBusinessName(iss.business_name);
        if (iss.has_logo) {
          setLoadingLogo(true);
          getPublicIssuerLogoBlobUrl(issuerId)
            .then((url) => { objectUrl = url; setLogoUrl(url); })
            .catch(() => setLogoUrl(null))
            .finally(() => setLoadingLogo(false));
        }
      })
      .catch(() => setNotFound(true))
      .finally(() => setLoadingIssuer(false));
    return () => { if (objectUrl) URL.revokeObjectURL(objectUrl); };
  }, [issuerId]);

  function handleFormSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setConfirming(true);
  }

  async function doRegister() {
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
      setConfirming(false);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo completar el registro");
      setConfirming(false);
    } finally {
      setSubmitting(false);
    }
  }

  function resetForm() {
    setTypeCode("13");
    setNumber("");
    setName("");
    setEmail("");
    setPhone("");
    setError(null);
    setConfirming(false);
    setDone(false);
  }

  const logoBlock = (loadingLogo || logoUrl) && (
    <AsyncImage src={logoUrl} loading={loadingLogo} alt={businessName ?? ""} className="h-14 w-14 rounded-lg p-1" />
  );

  return (
    <div className="flex min-h-screen items-start justify-center bg-(--bg-primary) px-4 pt-8 sm:items-center sm:pt-0">
      <Card className="w-full max-w-md">
        {loadingIssuer ? (
          <div className="flex min-h-32 items-center justify-center p-4">
            <Spinner className="h-5 w-5 text-(--text-muted)" />
          </div>
        ) : notFound ? (
          <div className="p-4">
            <Banner tone="danger">Este enlace de registro no es válido.</Banner>
          </div>
        ) : done ? (
          <div className="relative flex flex-col items-center gap-3 p-8 text-center">
            <button
              type="button"
              onClick={resetForm}
              className="absolute right-3 top-3 rounded p-1 text-(--text-muted) transition-colors hover:text-(--text-primary)"
              aria-label="Cerrar"
            >
              <X className="h-4 w-4" />
            </button>
            {logoBlock}
            <CheckCircle2 className="h-8 w-8 text-(--color-success-text)" />
            <div className="flex flex-col gap-1">
              <p className="text-sm font-semibold text-(--text-primary)">¡Listo!</p>
              <p className="text-xs text-(--text-secondary)">
                Quedaste registrado — {businessName} ya puede facturarte electrónicamente.
              </p>
            </div>
          </div>
        ) : confirming ? (
          <div className="flex flex-col items-center gap-4 p-8 text-center">
            {logoBlock}
            <div className="flex flex-col gap-1">
              <p className="text-sm font-semibold text-(--text-primary)">¿Estás seguro de registrarte?</p>
              <p className="text-xs text-(--text-secondary)">
                Se guardará tu información como cliente de <strong>{businessName}</strong>.
              </p>
            </div>
            {error && <Banner tone="danger">{error}</Banner>}
            <div className="flex w-full gap-3">
              <Button type="button" variant="secondary" className="flex-1" onClick={() => setConfirming(false)} disabled={submitting}>
                Volver
              </Button>
              <Button type="button" icon={<UserPlus className="h-3.5 w-3.5" />} loading={submitting} className="flex-1" onClick={doRegister}>
                Sí, registrarme
              </Button>
            </div>
          </div>
        ) : (
          <>
            <div className="flex flex-col items-center gap-3 border-b border-(--border-light) px-6 py-7 text-center">
              {(loadingLogo || logoUrl) && (
                <AsyncImage src={logoUrl} loading={loadingLogo} alt={businessName ?? ""} className="h-16 w-16 rounded-lg p-1" />
              )}
              <div className="flex flex-col gap-0.5">
                <p className="text-xs text-(--text-secondary)">Regístrate como cliente de</p>
                <h1 className="text-base font-semibold text-(--text-primary)">{businessName}</h1>
              </div>
            </div>
            <form className="flex flex-col gap-3 p-5" onSubmit={handleFormSubmit}>
              {error && <Banner tone="danger">{error}</Banner>}
              <div className="grid grid-cols-12 gap-3">
                <div className="col-span-5">
                  <Select label="Tipo de identificación" required value={typeCode} onChange={(e) => setTypeCode(e.target.value)}>
                    {IDENTIFICATION_TYPES.map((t) => (
                      <option key={t.code} value={t.code}>{t.name}</option>
                    ))}
                  </Select>
                </div>
                <div className="col-span-7">
                  <Input label="Número de identificación" required value={number} onChange={(e) => setNumber(e.target.value)} />
                </div>
                <div className="col-span-12">
                  <Input label="Nombre completo" required value={name} onChange={(e) => setName(e.target.value)} />
                </div>
                <div className="col-span-6">
                  <Input label="Correo electrónico" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
                </div>
                <div className="col-span-6">
                  <Input label="Teléfono (opcional)" value={phone} onChange={(e) => setPhone(e.target.value)} />
                </div>
              </div>
              <Button type="submit" icon={<UserPlus className="h-3.5 w-3.5" />} className="w-full">
                Registrarme
              </Button>
            </form>
          </>
        )}
      </Card>
    </div>
  );
}
