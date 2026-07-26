import { useEffect, useRef, useState } from "react";
import { X, Loader2, Mail } from "lucide-react";
import { Button } from "./Button";
import { Banner } from "./Banner";

interface Props {
  initialEmail: string;
  /** Si se provee, se llama al abrir para obtener el correo actualizado del catálogo. */
  fetchEmail?: () => Promise<string>;
  onSend: (to: string, cc: string[]) => Promise<void>;
  onClose: () => void;
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function SendEmailModal({ initialEmail, fetchEmail, onSend, onClose }: Props) {
  const [toInput, setToInput] = useState(initialEmail);
  const [ccInput, setCcInput] = useState("");
  const [resolving, setResolving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const toRef = useRef<HTMLInputElement>(null);

  // Obtener el correo actualizado del catálogo al abrir.
  useEffect(() => {
    if (!fetchEmail) return;
    setResolving(true);
    fetchEmail()
      .then((email) => { if (email) setToInput(email); })
      .catch(() => { /* fallback silencioso al initialEmail */ })
      .finally(() => setResolving(false));
  }, [fetchEmail]);

  async function handleSend() {
    const to = toInput.trim();
    if (!EMAIL_RE.test(to)) {
      setError("El correo del destinatario no es válido.");
      toRef.current?.focus();
      return;
    }

    const cc = ccInput
      .split(",")
      .map((e) => e.trim())
      .filter(Boolean);
    const invalid = cc.filter((e) => !EMAIL_RE.test(e));
    if (invalid.length > 0) {
      setError(`Correo${invalid.length > 1 ? "s" : ""} CC inválido${invalid.length > 1 ? "s" : ""}: ${invalid.join(", ")}`);
      return;
    }

    setError(null);
    setSending(true);
    try {
      await onSend(to, cc);
    } catch {
      // el llamador ya mostró el toast de error
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="w-full max-w-md rounded-lg border border-(--border-color) bg-(--bg-primary) p-5 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Mail className="h-4 w-4 text-(--accent-primary)" />
            <h2 className="text-sm font-semibold text-(--text-primary)">Enviar por correo</h2>
          </div>
          <button type="button" onClick={onClose} className="text-(--text-muted) hover:text-(--text-primary) transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-3">
          {/* Campo Para — editable */}
          <div>
            <label htmlFor="to-input" className="mb-1 flex items-center gap-1 text-xs font-medium text-(--text-secondary)">
              Para
              {resolving && <Loader2 className="h-3 w-3 animate-spin text-(--text-muted)" />}
            </label>
            <input
              id="to-input"
              ref={toRef}
              type="email"
              value={toInput}
              onChange={(e) => { setToInput(e.target.value); setError(null); }}
              onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); handleSend(); } }}
              disabled={resolving || sending}
              placeholder="correo@ejemplo.com"
              className="w-full rounded border border-(--border-color) bg-(--bg-primary) px-3 py-1.5 text-xs text-(--text-primary) transition-colors focus:outline-none focus:ring-1 focus:ring-(--accent-primary) disabled:opacity-50"
              autoFocus={!fetchEmail}
            />
          </div>

          {/* Campo CC */}
          <div>
            <label htmlFor="cc-input" className="mb-1 block text-xs font-medium text-(--text-secondary)">
              CC{" "}
              <span className="font-normal text-(--text-muted)">(opcional — separa múltiples correos con comas)</span>
            </label>
            <input
              id="cc-input"
              type="text"
              value={ccInput}
              onChange={(e) => { setCcInput(e.target.value); setError(null); }}
              onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); handleSend(); } }}
              disabled={sending}
              placeholder="correo@ejemplo.com, otro@ejemplo.com"
              className="w-full rounded border border-(--border-color) bg-(--bg-primary) px-3 py-1.5 text-xs text-(--text-primary) transition-colors focus:outline-none focus:ring-1 focus:ring-(--accent-primary) disabled:opacity-50"
            />
          </div>

          {error && <Banner tone="danger">{error}</Banner>}
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose} disabled={sending}>
            Cancelar
          </Button>
          <Button type="button" loading={sending} disabled={resolving} onClick={handleSend}>
            Enviar
          </Button>
        </div>
      </div>
    </div>
  );
}
