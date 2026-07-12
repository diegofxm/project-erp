import { AlertCircle, CheckCircle2, Info } from "lucide-react";
import { dianStatusLabel } from "../lib/dianStatus";

function iconForCode(code: string | undefined) {
  if (code === "00") return <CheckCircle2 className="mt-0.5 h-3.5 w-3.5 shrink-0 text-(--color-success-text)" />;
  if (code === "99") return <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-(--color-danger-text)" />;
  return <Info className="mt-0.5 h-3.5 w-3.5 shrink-0 text-(--color-info-text)" />;
}

export function DianStatusBlock({
  statusCode,
  description,
  message,
}: {
  statusCode: string | undefined;
  description: string | undefined;
  message: string | undefined;
}) {
  if (!statusCode && !description) return null;
  return (
    <div className="flex items-start gap-2 rounded border border-(--border-color) bg-(--bg-primary) p-3 text-xs">
      {iconForCode(statusCode)}
      <div>
        <p className="font-medium text-(--text-primary)">Estado DIAN: {dianStatusLabel(statusCode)}</p>
        {description && <p className="mt-1 text-(--text-secondary)">{description}</p>}
        {message && <p className="mt-1 text-(--text-secondary)">{message}</p>}
      </div>
    </div>
  );
}
