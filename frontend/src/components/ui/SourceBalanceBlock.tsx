import { ExternalLink } from "lucide-react";
import { Link } from "react-router";
import { formatCOP } from "../../lib/currency";
import type { Document } from "../../lib/types";
import { InfoTip } from "./InfoTip";

interface Props {
  doc: Document;           // FE o DS de origen
  pendingCents?: number;   // total vivo del formulario (>0 para mostrar)
  pendingTypeCode?: string; // "91" NC | "92" ND | "95" NA
  editingNoteId?: string;  // ID de la nota existente que se está editando (fusiona las dos filas)
}

const NOTE_LABELS: Record<string, string> = {
  "91": "Nota Crédito",
  "92": "Nota Débito",
  "95": "Nota de Ajuste",
};

export function SourceBalanceBlock({ doc, pendingCents, pendingTypeCode, editingNoteId }: Props) {
  const notes = doc.related_notes ?? [];
  const docLabel = doc.dian_document_type_code === "05" ? "Documento Soporte" : "Factura";
  const docUrl = doc.dian_document_type_code === "05"
    ? `/documents/support-documents/${doc.id}`
    : `/documents/invoices/${doc.id}`;

  // Saldo efectivo ya reconocido por la DIAN (solo notas accepted).
  const acceptedNet = doc.net_payable_cents ?? doc.totals.payable_cents;

  // Cuando se edita una nota existente, sacarla de la lista y fusionarla en una fila unificada.
  const editingNote = editingNoteId ? notes.find(n => n.id === editingNoteId) : undefined;
  const editingNoteIsAccepted = editingNote?.status === "accepted";

  // Notas aceptadas ajenas — si la nota en edición YA está aceptada, incluirla aquí (no en la fila de borrador).
  const displayNotes = notes.filter(
    n => n.status === "accepted" && (n.id !== editingNoteId || editingNoteIsAccepted),
  );

  // Fila de borrador solo cuando la nota en edición es un draft, o se está creando una nueva con valor.
  const editingTypeCode = pendingTypeCode ?? editingNote?.dian_document_type_code;
  const editingIsDebit = editingTypeCode === "92";
  const editingLabel = NOTE_LABELS[editingTypeCode ?? ""] ?? "Esta nota";
  const editingCents = (pendingCents ?? 0) > 0 ? (pendingCents ?? 0) : (editingNote?.payable_cents ?? 0);

  const showDraftRow =
    (!!editingNote && !editingNoteIsAccepted) ||
    (!editingNoteId && (pendingCents ?? 0) > 0);
  const draftRowLabel = editingNote
    ? `${editingLabel} (borrador en edición)`
    : `${editingLabel} (en edición)`;

  // Saldo proyectado (borrador pendiente) vs saldo contable (todo aceptado).
  const projectedCents = showDraftRow && editingCents > 0
    ? editingIsDebit
      ? acceptedNet + editingCents
      : acceptedNet - editingCents
    : null;

  const showFooter =
    projectedCents !== null ||
    (doc.net_payable_cents !== undefined && doc.net_payable_cents !== doc.totals.payable_cents);

  return (
    <div className="mt-3 rounded border border-(--color-info-border) bg-(--color-info-bg) p-3 text-xs">
      <div className="mb-2 flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <p className="font-semibold text-(--color-info-text)">
            {docLabel} {doc.prefix ?? ""}{doc.number ?? ""}
          </p>
          <InfoTip>
            Este panel es un <strong>visor didáctico de saldo</strong>: te muestra cómo
            quedaría el documento de origen después de aplicar las notas emitidas.
            {projectedCents !== null
              ? " El saldo proyectado incluye el borrador que estás editando y puede cambiar antes de confirmarlo."
              : " El saldo contable refleja solo notas ya aceptadas por la DIAN."}
          </InfoTip>
        </div>
        <Link
          to={docUrl}
          className="flex items-center gap-1 text-xs text-(--color-info-text) opacity-70 hover:opacity-100 hover:underline"
        >
          <ExternalLink className="h-3 w-3" />
          Ver {docLabel === "Documento Soporte" ? "DS" : "Factura"}
        </Link>
      </div>
      <div className="flex flex-col gap-1.5">
        {/* Total original */}
        <div className="flex items-center justify-between">
          <span className="text-(--color-info-text) opacity-80">Total original</span>
          <span className="font-mono text-(--color-info-text)">
            {formatCOP.format(doc.totals.payable_cents / 100)}
          </span>
        </div>

        {/* Otras notas (aceptadas o borradores que no son el que se edita) */}
        {displayNotes.map((note) => {
          const label = NOTE_LABELS[note.dian_document_type_code] ?? "Nota";
          const hasNumber = note.prefix || note.number;
          const isDebit = note.dian_document_type_code === "92";
          const isAccepted = note.status === "accepted";
          const ident = hasNumber
            ? ` ${note.prefix ?? ""}${note.number ?? ""}${!isAccepted ? " (pendiente)" : ""}`
            : isAccepted
              ? " (borrador)"
              : " (borrador pendiente)";
          return (
            <div
              key={note.id}
              className={`flex items-center justify-between ${!isAccepted ? "opacity-50" : ""}`}
            >
              <span className="text-(--color-info-text)">{label}{ident}</span>
              <span className={`font-mono ${isDebit ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>
                {isDebit ? "+" : "−"} {formatCOP.format(note.payable_cents / 100)}
              </span>
            </div>
          );
        })}

        {/* Fila de borrador en edición (solo cuando la nota no está aceptada aún) */}
        {showDraftRow && editingCents > 0 && (
          <div className="flex items-center justify-between italic">
            <span className="text-(--color-info-text)">{draftRowLabel}</span>
            <span className={`font-mono ${editingIsDebit ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>
              {editingIsDebit ? "+" : "−"} {formatCOP.format(editingCents / 100)}
            </span>
          </div>
        )}

        {/* Saldo */}
        {showFooter && (
          <div className="flex items-center justify-between border-t border-(--color-info-border) pt-1.5 font-semibold">
            <span className="text-(--color-info-text)">
              {projectedCents !== null ? "Saldo proyectado" : "Saldo contable"}
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
