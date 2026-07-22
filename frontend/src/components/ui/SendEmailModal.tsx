import { useState } from "react";
import { X } from "lucide-react";
import { Button } from "./Button";
import { Banner } from "./Banner";

interface Props {
  toEmail: string;
  onSend: (cc: string[]) => Promise<void>;
  onClose: () => void;
}

export function SendEmailModal({ toEmail, onSend, onClose }: Props) {
  const [ccInput, setCcInput] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);

  async function handleSend() {
    const cc = ccInput
      .split(",")
      .map((e) => e.trim())
      .filter(Boolean);

    const invalid = cc.filter((e) => !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(e));
    if (invalid.length > 0) {
      setError(`Correo${invalid.length > 1 ? "s" : ""} inválido${invalid.length > 1 ? "s" : ""}: ${invalid.join(", ")}`);
      return;
    }

    setError(null);
    setSending(true);
    try {
      await onSend(cc);
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
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-(--text-primary)">Enviar por correo</h2>
          <button type="button" onClick={onClose} className="text-(--text-muted) hover:text-(--text-primary) transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-3">
          <div>
            <p className="mb-1 text-xs font-medium text-(--text-secondary)">Para</p>
            <div className="rounded border border-(--border-color) bg-(--bg-secondary) px-3 py-1.5 text-xs text-(--text-muted)">
              {toEmail || "—"}
            </div>
          </div>

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
              placeholder="correo@ejemplo.com, otro@ejemplo.com"
              className="w-full rounded border border-(--border-color) bg-(--bg-primary) px-3 py-1.5 text-xs text-(--text-primary) transition-colors focus:outline-none focus:ring-1 focus:ring-(--accent-primary)"
              autoFocus
            />
          </div>

          {error && <Banner tone="danger">{error}</Banner>}
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose} disabled={sending}>
            Cancelar
          </Button>
          <Button type="button" loading={sending} onClick={handleSend}>
            Enviar
          </Button>
        </div>
      </div>
    </div>
  );
}
