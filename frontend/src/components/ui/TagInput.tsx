import { useState, type KeyboardEvent } from "react";
import { X } from "lucide-react";

interface TagInputProps {
  label?: string;
  values: string[];
  onChange: (values: string[]) => void;
  max?: number;
  placeholder?: string;
}

// Input de chips sin catálogo detrás (ej. códigos CIIU — DANE, no DIAN, ver
// cofacture/domain/types.go Party.IndustryClassificationCodes). Escribir un código + Enter
// agrega un chip; bloqueado en `max` chips.
export function TagInput({ label, values, onChange, max, placeholder }: TagInputProps) {
  const [draft, setDraft] = useState("");
  const atLimit = max !== undefined && values.length >= max;

  function addTag() {
    const code = draft.trim();
    if (!code || atLimit || values.includes(code)) return;
    onChange([...values, code]);
    setDraft("");
  }

  function removeTag(code: string) {
    onChange(values.filter((v) => v !== code));
  }

  function handleKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      addTag();
    }
  }

  return (
    <div className="flex flex-col gap-1">
      {label && <span className="text-xs font-medium text-(--text-secondary)">{label}</span>}
      <div className="flex w-full flex-wrap items-center gap-1.5 rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5">
        {values.map((code) => (
          <span
            key={code}
            className="flex items-center gap-1 rounded bg-(--bg-tertiary) px-1.5 py-0.5 text-xs font-mono text-(--text-primary)"
          >
            {code}
            <button type="button" onClick={() => removeTag(code)} className="text-(--text-muted) hover:text-(--color-danger)">
              <X className="h-3 w-3" />
            </button>
          </span>
        ))}
        {!atLimit && (
          <input
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={handleKeyDown}
            onBlur={addTag}
            placeholder={placeholder}
            className="min-w-[6rem] flex-1 bg-transparent text-xs text-(--text-primary) outline-none placeholder:text-(--text-muted)"
          />
        )}
      </div>
      {atLimit && <span className="text-xs text-(--text-muted)">Máximo {max} códigos.</span>}
    </div>
  );
}
