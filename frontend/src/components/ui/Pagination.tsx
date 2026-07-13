import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "./Button";

interface PaginationProps {
  offset: number;
  count: number;
  hasNext: boolean;
  loading?: boolean;
  onPrev: () => void;
  onNext: () => void;
}

export function Pagination({ offset, count, hasNext, loading, onPrev, onNext }: PaginationProps) {
  const hasPrev = offset > 0;
  if (!hasPrev && !hasNext) return null;
  return (
    <div className="mt-3 flex items-center justify-between">
      <span className="text-xs text-(--text-secondary)">
        Mostrando {offset + 1}–{offset + count}
      </span>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="secondary"
          icon={<ChevronLeft className="h-3.5 w-3.5" />}
          disabled={!hasPrev || loading}
          onClick={onPrev}
        >
          Anterior
        </Button>
        <Button
          type="button"
          variant="secondary"
          icon={<ChevronRight className="h-3.5 w-3.5" />}
          disabled={!hasNext || loading}
          loading={loading}
          onClick={onNext}
        >
          Siguiente
        </Button>
      </div>
    </div>
  );
}
