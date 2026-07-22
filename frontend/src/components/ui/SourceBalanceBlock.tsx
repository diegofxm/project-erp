import { formatCOP } from "../../lib/currency";
import type { Document } from "../../lib/types";

interface Props {
  doc: Document;           // FE o DS de origen
  pendingCents?: number;   // total vivo del formulario (>0 para mostrar)
  pendingTypeCode?: string; // "91" NC | "92" ND | "95" NA
}

const NOTE_LABELS: Record<string, string> = {
  "91": "Nota Crédito",
  "92": "Nota Débito",
  "95": "Nota de Ajuste",
};

// Muestra el saldo disponible de la FE/DS de origen mientras se redacta una NC/ND/NA.
// Se actualiza en vivo (pendingCents) a medida que el operador edita las líneas del formulario.
export function SourceBalanceBlock({ doc, pendingCents, pendingTypeCode }: Props) {
  const notes = doc.related_notes ?? [];
  const docLabel = doc.dian_document_type_code === "05" ? "Documento Soporte" : "Factura";
  const isDebitPending = pendingTypeCode === "92";

  // Saldo efectivo ya reconocido por la DIAN (solo notas accepted).
  const acceptedNet = doc.net_payable_cents ?? doc.totals.payable_cents;

  // Saldo proyectado incluyendo la nota que se está redactando.
  const hasPending = pendingCents !== undefined && pendingCents > 0;
  const projectedCents = hasPending
    ? isDebitPending
      ? acceptedNet + pendingCents
      : acceptedNet - pendingCents
    : null;

  return (
    <div className="mt-3 rounded border border-(--color-info-border) bg-(--color-info-bg) p-3 text-xs">
      <p className="mb-2 font-semibold text-(--color-info-text)">
        {docLabel} {doc.prefix ?? ""}{doc.number ?? ""}
      </p>
      <div className="flex flex-col gap-1.5">
        {/* Total original */}
        <div className="flex items-center justify-between">
          <span className="text-(--color-info-text) opacity-80">Total original</span>
          <span className="font-mono text-(--color-info-text)">
            {formatCOP.format(doc.totals.payable_cents / 100)}
          </span>
        </div>

        {/* Notas ya existentes */}
        {notes.map((note) => {
          const label = NOTE_LABELS[note.dian_document_type_code] ?? "Nota";
          const ident = note.prefix || note.number
            ? ` ${note.prefix ?? ""}${note.number ?? ""}`
            : " (borrador)";
          const isDebit = note.dian_document_type_code === "92";
          const isAccepted = note.status === "accepted";
          return (
            <div
              key={note.id}
              className={`flex items-center justify-between ${!isAccepted ? "opacity-50" : ""}`}
            >
              <span className="text-(--color-info-text)">
                {label}{ident}{!isAccepted ? " (pendiente)" : ""}
              </span>
              <span className={`font-mono ${isDebit ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>
                {isDebit ? "+" : "−"} {formatCOP.format(note.payable_cents / 100)}
              </span>
            </div>
          );
        })}

        {/* Nota que se está redactando ahora */}
        {hasPending && (
          <div className="flex items-center justify-between italic">
            <span className="text-(--color-info-text)">
              {NOTE_LABELS[pendingTypeCode ?? ""] ?? "Esta nota"} (en edición)
            </span>
            <span className={`font-mono ${isDebitPending ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>
              {isDebitPending ? "+" : "−"} {formatCOP.format(pendingCents / 100)}
            </span>
          </div>
        )}

        {/* Saldo proyectado — solo cuando hay algo en edición o notas aceptadas cambian el total */}
        {(hasPending || (doc.net_payable_cents !== undefined && doc.net_payable_cents !== doc.totals.payable_cents)) && (
          <div className="flex items-center justify-between border-t border-(--color-info-border) pt-1.5 font-semibold">
            <span className="text-(--color-info-text)">
              {hasPending ? "Saldo proyectado" : "Saldo efectivo"}
            </span>
            <span className="font-mono text-(--color-info-text)">
              {formatCOP.format((projectedCents ?? acceptedNet) / 100)}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
