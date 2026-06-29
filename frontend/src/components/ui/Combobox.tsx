import { useEffect, useRef, useState, type KeyboardEvent } from "react";

export interface ComboboxOption {
  value: string;
  label: string;
}

interface ComboboxProps {
  label?: string;
  value: string;
  onChange: (value: string) => void;
  options: ComboboxOption[];
  placeholder?: string;
  disabled?: boolean;
}

// Selector con búsqueda en memoria — mismo look que Select.tsx/Input.tsx, pero filtra
// `options` por substring (case-insensitive, sobre `label`) a medida que se escribe. Al elegir
// una opción, onChange recibe solo `value` (el código) — igual contrato que un <select>
// nativo, nunca queda la descripción completa en el estado del formulario. Pensado para
// catálogos chicos-a-medianos que cargan completos una sola vez (unit_measures, CIIU) — sin
// debounce ni búsqueda remota, eso no hace falta a esa escala.
export function Combobox({ label, value, onChange, options, placeholder, disabled }: ComboboxProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [highlighted, setHighlighted] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);

  const selected = options.find((o) => o.value === value);
  const filtered =
    query.trim() === "" ? options : options.filter((o) => o.label.toLowerCase().includes(query.trim().toLowerCase()));

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
        setQuery("");
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  function handleSelect(option: ComboboxOption) {
    onChange(option.value);
    setQuery("");
    setOpen(false);
  }

  function handleKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (!open) {
      if (e.key === "ArrowDown" || e.key === "Enter") setOpen(true);
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setHighlighted((h) => Math.min(h + 1, filtered.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setHighlighted((h) => Math.max(h - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const option = filtered[highlighted];
      if (option) handleSelect(option);
    } else if (e.key === "Escape") {
      setOpen(false);
      setQuery("");
    }
  }

  return (
    <div className="relative flex flex-col gap-1" ref={containerRef}>
      {label && <span className="text-xs font-medium text-(--text-secondary)">{label}</span>}
      <input
        value={open ? query : selected?.label ?? ""}
        onChange={(e) => {
          setQuery(e.target.value);
          setHighlighted(0);
          if (!open) setOpen(true);
        }}
        onFocus={() => {
          setOpen(true);
          setQuery("");
          setHighlighted(0);
        }}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        disabled={disabled}
        className="w-full rounded border border-(--border-color) bg-(--bg-primary) px-3 py-1.5 text-xs text-(--text-primary) placeholder:text-(--text-muted) disabled:cursor-not-allowed disabled:opacity-60"
      />
      {open && filtered.length > 0 && (
        <div className="absolute top-full z-20 mt-1 max-h-56 w-full overflow-auto rounded border border-(--border-light) bg-(--bg-secondary) shadow-lg">
          {filtered.map((option, i) => (
            <button
              key={option.value}
              type="button"
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => handleSelect(option)}
              className={`block w-full truncate px-3 py-1.5 text-left text-xs hover:bg-(--bg-hover) ${
                i === highlighted ? "bg-(--bg-hover) text-(--text-primary)" : "text-(--text-secondary)"
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
      )}
      {open && filtered.length === 0 && (
        <div className="absolute top-full z-20 mt-1 w-full rounded border border-(--border-light) bg-(--bg-secondary) px-3 py-1.5 text-xs text-(--text-muted) shadow-lg">
          Sin resultados
        </div>
      )}
    </div>
  );
}
