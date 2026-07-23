import { Info } from "lucide-react";
import { useEffect, useRef, useState } from "react";

interface Props {
  children: React.ReactNode;
  className?: string;
}

/**
 * Icono ⓘ con popover explicativo al hacer clic.
 * Abre a la derecha por defecto (flujo natural de lectura); si el espacio
 * a la derecha es insuficiente (<260px), se voltea automáticamente a la izquierda.
 *
 * Uso: <InfoTip>Texto explicativo.</InfoTip>
 */
export function InfoTip({ children, className }: Props) {
  const [open, setOpen] = useState(false);
  const [flipLeft, setFlipLeft] = useState(false);
  const btnRef = useRef<HTMLButtonElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Detectar si hay espacio a la derecha al abrir.
  function handleOpen() {
    if (!open && btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect();
      setFlipLeft(window.innerWidth - rect.right < 260);
    }
    setOpen((o) => !o);
  }

  // Cerrar al clic fuera.
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

  // Posición del popover: derecha o izquierda del ícono, centrado verticalmente.
  const popoverPos = flipLeft
    ? "right-full mr-2 top-1/2 -translate-y-1/2"
    : "left-full ml-2 top-1/2 -translate-y-1/2";

  // Flecha apuntando hacia el ícono.
  const arrowPos = flipLeft
    ? "left-full top-1/2 -translate-y-1/2 border-l-(--bg-primary) border-y-transparent border-r-transparent"
    : "right-full top-1/2 -translate-y-1/2 border-r-(--bg-primary) border-y-transparent border-l-transparent";

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
          className={`absolute ${popoverPos} z-50 w-56 rounded border border-(--border-color) bg-(--bg-primary) p-2.5 shadow-lg`}
          role="tooltip"
        >
          <p className="text-[11px] leading-relaxed text-(--text-secondary)">{children}</p>
          <span className={`absolute ${arrowPos} border-4`} />
        </div>
      )}
    </div>
  );
}
