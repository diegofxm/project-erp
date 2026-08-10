import { useEffect, useId, useRef, useState, type KeyboardEvent } from "react";

export interface ComboboxOption {
  value: string;
  label: string;
}

interface ComboboxProps {
  label?: string;
  // ariaLabel: nombre accesible sin renderizar un <span> visible -- para usos compactos (ej. una
  // celda de tabla) donde la columna ya da contexto visual pero un lector de pantalla igual
  // necesita un nombre propio para el control. Si se pasan ambos, `label` gana (aria-labelledby
  // sobre el <span> visible es más específico que un aria-label plano).
  ariaLabel?: string;
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
export function Combobox({ label, ariaLabel, value, onChange, options, placeholder, disabled }: ComboboxProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [highlighted, setHighlighted] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);
  // Semántica ARIA de combobox (ver docs/auditorias/2026-08-09/05-frontend.md punto 24): el foco
  // real del DOM se queda en el <input> (igual que antes); role="option"/aria-activedescendant
  // simulan la navegación por las opciones para lectores de pantalla sin mover el foco.
  const baseId = useId();
  const labelId = `${baseId}-label`;
  const listboxId = `${baseId}-listbox`;
  const optionId = (optionValue: string) => `${baseId}-option-${optionValue}`;

  const selected = options.find((o) => o.value === value);
  const filtered =
    query.trim() === "" ? options : options.filter((o) => o.label.toLowerCase().includes(query.trim().toLowerCase()));
  const activeOption = open ? filtered[highlighted] : undefined;

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
      {label && (
        <span id={labelId} className="text-xs font-medium text-(--text-secondary)">
          {label}
        </span>
      )}
      <input
        role="combobox"
        aria-expanded={open}
        aria-controls={listboxId}
        aria-autocomplete="list"
        aria-activedescendant={activeOption ? optionId(activeOption.value) : undefined}
        aria-labelledby={label ? labelId : undefined}
        aria-label={!label ? ariaLabel : undefined}
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
      {open && (
        <div
          id={listboxId}
          role="listbox"
          aria-labelledby={label ? labelId : undefined}
          className="absolute top-full z-20 mt-1 max-h-56 w-full overflow-auto rounded border border-(--border-light) bg-(--bg-secondary) shadow-lg"
        >
          {filtered.length === 0 ? (
            <div className="px-3 py-1.5 text-xs text-(--text-muted)">Sin resultados</div>
          ) : (
            filtered.map((option, i) => (
              <button
                key={option.value}
                id={optionId(option.value)}
                role="option"
                aria-selected={i === highlighted}
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => handleSelect(option)}
                className={`block w-full truncate px-3 py-1.5 text-left text-xs hover:bg-(--bg-hover) ${
                  i === highlighted ? "bg-(--bg-hover) text-(--text-primary)" : "text-(--text-secondary)"
                }`}
              >
                {option.label}
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}
