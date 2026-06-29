import { Spinner } from "./Spinner";

interface AsyncImageProps {
  src: string | null;
  loading: boolean;
  alt: string;
  className?: string; // debe incluir el tamaño (h-X w-X) — se aplica igual al recuadro de carga y a la imagen real
}

// Regla general para cualquier imagen que se trae de forma asíncrona (logo del emisor, hoy en
// LogoForm.tsx/PublicCustomerRegisterPage.tsx) — mientras loading es true se ve un Spinner
// dentro de un recuadro del mismo tamaño/borde que tendría la imagen, nunca un hueco en blanco
// que de repente "salta" cuando llega el blob. loading=false y src=null significa "no hay
// imagen" (ej. el emisor no configuró logo) — ahí no se renderiza nada.
export function AsyncImage({ src, loading, alt, className = "" }: AsyncImageProps) {
  if (loading) {
    return (
      <div className={`flex items-center justify-center rounded border border-(--border-color) bg-(--bg-primary) ${className}`}>
        <Spinner className="h-4 w-4 text-(--text-muted)" />
      </div>
    );
  }
  if (!src) return null;
  return <img src={src} alt={alt} className={`object-contain ${className}`} />;
}
