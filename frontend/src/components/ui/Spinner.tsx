import { Loader2 } from "lucide-react";

// Spinner de carga (sección 14 del design system: "iconos RefreshCw/Loader2 con animate-spin")
// — sin color propio por defecto, hereda currentColor del contenedor (texto blanco dentro de
// un Button primario, --text-muted cuando se centra solo dentro de una tarjeta).
export function Spinner({ className = "h-4 w-4" }: { className?: string }) {
  return <Loader2 className={`animate-spin ${className}`} />;
}
