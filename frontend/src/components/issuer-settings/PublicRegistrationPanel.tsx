import { useEffect, useState } from "react";
import QRCode from "qrcode";
import { Check, Copy, Download } from "lucide-react";
import { useAuth } from "../../context/AuthContext";
import { Button } from "../ui/Button";

// Panel de autorregistro de clientes por QR (patrón D1 y similares, ver
// docs/apidian-architecture.md sección 9.41) — el emisor imprime este QR/link en su mostrador;
// quien lo escanea llega a /r/:issuerId (página pública, sin sesión) y se autorregistra como
// adquiriente sin que nadie tenga que digitarlo por él. El QR se genera en el navegador
// (paquete qrcode) — no hace falta pedirle nada al backend más allá del link en sí.
export function PublicRegistrationPanel() {
  const { activeIssuer } = useAuth();
  const [qrDataURL, setQrDataURL] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const link = activeIssuer ? `${window.location.origin}/r/${activeIssuer.id}` : "";

  useEffect(() => {
    if (!link) return;
    QRCode.toDataURL(link, { width: 240, margin: 1 })
      .then(setQrDataURL)
      .catch(() => setQrDataURL(null));
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
    <div className="flex flex-col gap-3 rounded border border-(--border-color) p-4">
      <h2 className="text-xs font-semibold text-(--text-primary)">Autorregistro de clientes</h2>
      <p className="text-xs text-(--text-secondary)">
        Comparte este código QR o enlace en tu mostrador — quien lo escanee podrá registrarse como cliente sin que tengas que
        digitar sus datos.
      </p>
      <div className="flex items-center gap-4">
        {qrDataURL && <img src={qrDataURL} alt="QR de autorregistro" className="h-32 w-32 rounded border border-(--border-color)" />}
        <div className="flex flex-1 flex-col gap-2">
          <p className="break-all rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 font-mono text-xs text-(--text-secondary)">
            {link}
          </p>
          <div className="flex gap-2">
            <Button type="button" variant="secondary" icon={copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />} onClick={handleCopy}>
              {copied ? "Copiado" : "Copiar enlace"}
            </Button>
            <Button type="button" variant="secondary" icon={<Download className="h-3.5 w-3.5" />} onClick={handleDownload} disabled={!qrDataURL}>
              Descargar QR
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
