import { Info } from "lucide-react";
import { useEffect, useRef, useState } from "react";

interface Props {
  children: React.ReactNode;
  className?: string;
}

/**
 * Ícono ⓘ con popover explicativo al hacer clic.
 * Abre debajo del ícono por defecto (recuadro horizontal, más ancho que alto);
 * si no hay espacio abajo se voltea arriba. Alineación izquierda o derecha
 * según el espacio disponible.
 */
export function InfoTip({ children, className }: Props) {
  const [open, setOpen] = useState(false);
  const [flipUp, setFlipUp] = useState(false);
  const [flipLeft, setFlipLeft] = useState(false);
  const btnRef = useRef<HTMLButtonElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const POPOVER_W = 320;
  const POPOVER_H_ESTIMATE = 160; // estimado conservador

  function handleOpen() {
    if (!open && btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect();
      setFlipUp(window.innerHeight - rect.bottom < POPOVER_H_ESTIMATE);
      setFlipLeft(window.innerWidth - rect.left < POPOVER_W);
    }
    setOpen((o) => !o);
  }

  useEffect(() => {
    if (!open) return;
    function onPointer(e: PointerEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("pointerdown", onPointer);
    return () => document.removeEventListener("pointerdown", onPointer);
  }, [open]);

  // Posición vertical: debajo o arriba del ícono.
  const vertPos = flipUp ? "bottom-full mb-2" : "top-full mt-2";
  // Alineación horizontal: desde la izquierda del ícono o anclado a la derecha.
  const horizPos = flipLeft ? "right-0" : "left-0";

  // Flecha
  const arrowV = flipUp
    ? "top-full border-t-(--bg-primary) border-x-transparent border-b-transparent"
    : "bottom-full border-b-(--bg-primary) border-x-transparent border-t-transparent";
  const arrowH = flipLeft ? "right-1.5" : "left-1.5";

  return (
    <div ref={containerRef} className={`relative inline-flex items-center ${className ?? ""}`}>
      <button
        ref={btnRef}
        type="button"
        onClick={handleOpen}
        aria-label="Más información"
        className="flex items-center rounded-full text-(--text-muted) transition-colors hover:text-(--accent-primary) focus:outline-none"
      >
        <Info className="h-3 w-3" />
      </button>

      {open && (
        <div
          className={`absolute ${vertPos} ${horizPos} z-50 w-80 rounded border border-(--border-color) bg-(--bg-primary) p-3 shadow-lg`}
          role="tooltip"
        >
          <p className="text-[11px] leading-relaxed text-(--text-secondary)">{children}</p>
          <span className={`absolute ${arrowV} ${arrowH} border-4`} />
        </div>
      )}
    </div>
  );
}
