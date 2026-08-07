import { Info } from "lucide-react";
import { useEffect, useRef, useState } from "react";

interface Props {
  children: React.ReactNode;
}

/**
 * Ícono ⓘ con popover explicativo al hacer clic. Único componente de infotip del proyecto — todo
 * el estilo visual (tamaño de letra, color, ancho, padding, borde) vive aquí, a propósito, sin
 * prop `className` para el popover: lo único que debe cambiar de una página a otra es el texto
 * (`children`), nunca la apariencia — así los ~11 usos en el proyecto quedan siempre idénticos
 * entre sí, sin que una página se desincronice del resto. Para resaltar una palabra dentro del
 * texto usa <strong>, que ya trae su propio estilo (ver abajo) — no metas spans con clases sueltas.
 *
 * Abre debajo del ícono por defecto (recuadro horizontal, más ancho que alto);
 * si no hay espacio abajo se voltea arriba. Alineación izquierda o derecha
 * según el espacio disponible. Sin flecha/triángulo apuntando al ícono — con eso el borde del
 * recuadro queda continuo en los cuatro lados (antes el triángulo, dibujado con bordes de
 * colores, dejaba un pequeño corte justo donde se unía con el borde del recuadro).
 */
export function InfoTip({ children }: Props) {
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

  return (
    <div ref={containerRef} className="relative inline-flex items-center">
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
          <p className="text-[11px] leading-relaxed text-(--text-secondary) [&_strong]:font-semibold [&_strong]:text-(--text-primary)">
            {children}
          </p>
        </div>
      )}
    </div>
  );
}
