import { useEffect, useState } from "react";
import QRCode from "qrcode";
import { Check, Copy, Download } from "lucide-react";
import { useAuth } from "../../context/AuthContext";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";
import { Spinner } from "../ui/Spinner";

// Panel de autorregistro de clientes por QR (patrón D1 y similares, ver
// docs/apidian-architecture.md sección 9.41) — la empresa imprime este QR/link en su mostrador;
// quien lo escanea llega a /r/:companyId (página pública, sin sesión) y se autorregistra como
// adquiriente sin que nadie tenga que digitarlo por él. El QR se genera en el navegador
// (paquete qrcode) — no hace falta pedirle nada al backend más allá del link en sí.
export function PublicRegistrationPanel() {
  const { activeCompany } = useAuth();
  const [qrDataURL, setQrDataURL] = useState<string | null>(null);
  const [loadingQR, setLoadingQR] = useState(true);
  const [copied, setCopied] = useState(false);

  const link = activeCompany ? `${window.location.origin}/r/${activeCompany.id}` : "";

  useEffect(() => {
    if (!link) {
      setLoadingQR(false);
      return;
    }
    setLoadingQR(true);
    QRCode.toDataURL(link, { width: 240, margin: 1 })
      .then(setQrDataURL)
      .catch(() => setQrDataURL(null))
      .finally(() => setLoadingQR(false));
  }, [link]);

  async function handleCopy() {
    await navigator.clipboard.writeText(link);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  function handleDownload() {
    if (!qrDataURL) return;
    const a = document.createElement("a");
    a.href = qrDataURL;
    a.download = "qr-autorregistro-clientes.png";
    a.click();
  }

  return (
    <Card className="flex flex-col gap-3 p-4">
      <h2 className="text-xs font-semibold text-(--text-primary)">Autorregistro de clientes</h2>
      <p className="text-xs text-(--text-secondary)">
        Comparte este código QR o enlace en tu mostrador — quien lo escanee podrá registrarse como cliente sin que tengas que
        digitar sus datos.
      </p>
      <div className="flex items-center gap-4">
        {loadingQR ? (
          <div className="flex h-32 w-32 items-center justify-center rounded border border-(--border-color)">
            <Spinner className="h-5 w-5 text-(--text-muted)" />
          </div>
        ) : (
          qrDataURL && <img src={qrDataURL} alt="QR de autorregistro" className="h-32 w-32 rounded border border-(--border-color)" />
        )}
        <div className="flex flex-1 flex-col gap-2">
          <div className="relative">
            <p className="break-all rounded border border-(--border-color) bg-(--bg-primary) py-1.5 pl-2 pr-8 font-mono text-xs text-(--text-secondary)">
              {link}
            </p>
            <button
              type="button"
              onClick={handleCopy}
              title={copied ? "Copiado" : "Copiar enlace"}
              aria-label={copied ? "Copiado" : "Copiar enlace"}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 transition-colors text-(--text-muted) hover:text-(--text-primary)"
            >
              {copied
                ? <Check className="h-3.5 w-3.5 text-(--color-success-text)" />
                : <Copy className="h-3.5 w-3.5" />}
            </button>
          </div>
          <div className="flex gap-2">
            <Button type="button" variant="secondary" icon={<Download className="h-3.5 w-3.5" />} onClick={handleDownload} disabled={!qrDataURL}>
              Descargar QR
            </Button>
          </div>
        </div>
      </div>
    </Card>
  );
}
