import { AlertTriangle } from "lucide-react";
import { Button } from "./Button";
import { Card } from "./Card";

export interface ConfirmRequest {
  message: string;
  title?: string;
  tone?: "danger" | "default";
  confirmLabel?: string;
}

interface ConfirmDialogProps extends ConfirmRequest {
  onConfirm: () => void;
  onCancel: () => void;
}

// Modal propio para confirmaciones — reemplaza window.confirm (diálogo nativo del navegador,
// sin estilo propio) en toda acción destructiva. Mismo Card/Button que el resto del dashboard,
// nada nuevo que aprender visualmente.
export function ConfirmDialog({ message, title, tone = "default", confirmLabel, onConfirm, onCancel }: ConfirmDialogProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onCancel}>
      <Card className="w-full max-w-sm" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-start gap-3 p-4">
          {tone === "danger" && <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0 text-(--color-danger-text)" />}
          <div className="flex flex-col gap-1">
            {title && <h2 className="text-sm font-semibold text-(--text-primary)">{title}</h2>}
            <p className="text-xs text-(--text-secondary)">{message}</p>
          </div>
        </div>
        <div className="flex justify-end gap-2 border-t border-(--border-light) p-3">
          <Button type="button" variant="secondary" onClick={onCancel}>
            Cancelar
          </Button>
          <Button type="button" variant={tone === "danger" ? "danger" : "primary"} onClick={onConfirm}>
            {confirmLabel ?? (tone === "danger" ? "Eliminar" : "Confirmar")}
          </Button>
        </div>
      </Card>
    </div>
  );
}
