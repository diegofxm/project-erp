import { Info } from "lucide-react";
import { useEffect, useRef, useState } from "react";

interface Props {
  children: React.ReactNode;
  /** Dirección del popover: "up" (default) o "down" */
  direction?: "up" | "down";
  className?: string;
}

/**
 * Icono ⓘ que abre un popover con explicación al hacer clic.
 * Úsalo para dar contexto didáctico sin ocupar espacio en el layout:
 *   <InfoTip>Texto explicativo aquí.</InfoTip>
 */
export function InfoTip({ children, direction = "up", className }: Props) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function onPointer(e: PointerEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("pointerdown", onPointer);
    return () => document.removeEventListener("pointerdown", onPointer);
  }, [open]);

  const popoverPos =
    direction === "down"
      ? "top-full mt-1.5 left-1/2 -translate-x-1/2"
      : "bottom-full mb-1.5 left-1/2 -translate-x-1/2";

  const arrowPos =
    direction === "down"
      ? "bottom-full left-1/2 -translate-x-1/2 border-b-(--bg-primary) border-x-transparent border-t-transparent"
      : "top-full left-1/2 -translate-x-1/2 border-t-(--bg-primary) border-x-transparent border-b-transparent";

  return (
    <div ref={ref} className={`relative inline-flex items-center ${className ?? ""}`}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-label="Más información"
        className="flex items-center rounded-full text-(--text-muted) transition-colors hover:text-(--accent-primary) focus:outline-none"
      >
        <Info className="h-3 w-3" />
      </button>

      {open && (
        <div
          className={`absolute ${popoverPos} z-50 w-60 rounded border border-(--border-color) bg-(--bg-primary) p-2.5 shadow-lg`}
          role="tooltip"
        >
          <p className="text-[11px] leading-relaxed text-(--text-secondary)">{children}</p>
          {/* flecha */}
          <span className={`absolute ${arrowPos} border-4`} />
        </div>
      )}
    </div>
  );
}
