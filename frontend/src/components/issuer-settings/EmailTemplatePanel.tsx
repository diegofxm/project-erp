import { useState, type FormEvent } from "react";
import { Mail } from "lucide-react";
import { useAuth } from "../../context/AuthContext";
import { useToast } from "../../context/ToastContext";
import { ApiError } from "../../lib/apiClient";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";

const PLACEHOLDERS = [
  { key: "{nombre_cliente}", desc: "Nombre del receptor" },
  { key: "{numero_documento}", desc: "Número del documento (ej. PREF-001)" },
  { key: "{nombre_empresa}", desc: "Razón social del emisor" },
  { key: "{tipo_documento}", desc: "\"factura electrónica\", \"nota crédito\" o \"nota débito\"" },
];

export function EmailTemplatePanel() {
  const { activeIssuer, updateIssuer } = useAuth();
  const toast = useToast();
  const [body, setBody] = useState(activeIssuer?.email_body_template ?? "");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      await updateIssuer({ email_body_template: body });
      toast.success(body.trim() ? "Plantilla guardada." : "Plantilla restablecida al texto predeterminado.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo guardar la plantilla");
    } finally {
      setLoading(false);
    }
  }

  function handleReset() {
    setBody("");
  }

  return (
    <Card className="flex flex-col gap-3 p-4">
      <h2 className="flex items-center gap-1.5 text-xs font-semibold text-(--text-primary)">
        <Mail className="h-3.5 w-3.5 shrink-0 text-(--accent-primary)" />
        Cuerpo del correo al cliente
      </h2>
      <p className="text-xs text-(--text-secondary)">
        Texto que se envía al cliente junto con el ZIP del documento. Deja vacío para usar el mensaje predeterminado.
        Puedes insertar variables entre llaves:
      </p>
      <div className="flex flex-wrap gap-x-4 gap-y-1">
        {PLACEHOLDERS.map(({ key, desc }) => (
          <div key={key} className="flex items-center gap-1.5">
            <code className="rounded bg-(--bg-tertiary) px-1 py-0.5 font-mono text-[10px] text-(--accent-primary)">{key}</code>
            <span className="text-[10px] text-(--text-muted)">{desc}</span>
          </div>
        ))}
      </div>
      <form className="flex flex-col gap-3" onSubmit={handleSubmit}>
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={6}
          placeholder={
            "Hola {nombre_cliente},\n\nAdjuntamos tu {tipo_documento} No. {numero_documento}, emitida por {nombre_empresa}.\n\n..."
          }
          className="w-full resize-y rounded border border-(--border-color) bg-(--bg-primary) px-3 py-2 font-mono text-xs text-(--text-primary) placeholder:text-(--text-muted) focus:outline-none focus:ring-1 focus:ring-(--accent-primary)"
        />
        <div className="flex items-center gap-2">
          <Button type="submit" loading={loading}>
            Guardar
          </Button>
          {body.trim() && (
            <Button type="button" variant="secondary" onClick={handleReset}>
              Restablecer predeterminado
            </Button>
          )}
        </div>
      </form>
    </Card>
  );
}
