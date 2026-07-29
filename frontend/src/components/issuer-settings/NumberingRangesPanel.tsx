import { useEffect, useState } from "react";
import { Archive, Ban, ChevronDown, ChevronRight, Download, Plus, RotateCcw, X } from "lucide-react";
import { Pagination } from "../ui/Pagination";
import { listDianDocumentTypes } from "../../lib/catalogs";
import { activateNumberingRange, createNumberingRange, deactivateNumberingRange, listNumberingRanges } from "../../lib/numberingRanges";
import { getDianNumberingRanges } from "../../lib/dianVerification";
import { ApiError } from "../../lib/apiClient";
import { useCatalog } from "../../lib/useCatalog";
import { useAuth } from "../../context/AuthContext";
import { useConfirm } from "../../context/ConfirmContext";
import { useToast } from "../../context/ToastContext";
import type { CreateNumberingRangePayload, DianRange, NumberingRange, NumberingRangeStatus } from "../../lib/types";
import { Button } from "../ui/Button";
import { Banner } from "../ui/Banner";
import { Card } from "../ui/Card";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { Spinner } from "../ui/Spinner";
import { NumberingRangeForm } from "./NumberingRangeForm";

const STATUS_LABELS: Record<NumberingRangeStatus, string> = {
  active: "Activo",
  expired: "Vencido",
  exhausted: "Agotado",
  inactive: "Inactivo",
};
const STATUS_CLASSES: Record<NumberingRangeStatus, string> = {
  active: "bg-(--color-success-bg) text-(--color-success-text)",
  expired: "bg-(--color-danger-bg) text-(--color-danger-text)",
  exhausted: "bg-(--color-info-bg) text-(--color-info-text)",
  inactive: "bg-(--bg-tertiary) text-(--text-secondary)",
};

const RANGES_PAGE_SIZE = 5;

function wouldBeUsable(r: NumberingRange): boolean {
  if (new Date(r.valid_to) < new Date()) return false;
  if (r.range_to !== undefined && r.current_number >= r.range_to) return false;
  return true;
}

function isExpiringSoon(r: NumberingRange): boolean {
  if (r.status !== "active") return false;
  const daysLeft = (new Date(r.valid_to).getTime() - Date.now()) / (1000 * 60 * 60 * 24);
  return daysLeft <= 30;
}

function RangesTable({ rows, docTypeName, onDeactivate, onArchive, onActivate }: {
  rows: NumberingRange[];
  docTypeName: (code: string) => string;
  onDeactivate: (r: NumberingRange) => void;
  onArchive: (r: NumberingRange) => void;
  onActivate: (r: NumberingRange) => void;
}) {
  const [offset, setOffset] = useState(0);
  const page = rows.slice(offset, offset + RANGES_PAGE_SIZE);
  const hasNext = offset + RANGES_PAGE_SIZE < rows.length;

  return (
    <>
      <div className="overflow-x-auto rounded border border-(--border-color)">
        <table className="w-full text-left text-xs">
          <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
            <tr>
              <th className="px-3 py-2 font-medium">Tipo</th>
              <th className="px-3 py-2 font-medium">Resolución</th>
              <th className="px-3 py-2 font-medium">Actual</th>
              <th className="px-3 py-2 font-medium">Vence</th>
              <th className="px-3 py-2 font-medium">Ambiente</th>
              <th className="px-3 py-2 font-medium">Estado</th>
              <th className="px-3 py-2" />
            </tr>
          </thead>
          <tbody>
            {page.map((r, i) => (
              <tr key={r.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                <td className="whitespace-nowrap px-3 py-2 font-medium text-(--text-primary)">{docTypeName(r.dian_document_type_code)}</td>
                <td className="whitespace-nowrap px-3 py-2 font-mono text-(--text-secondary)">{r.prefix} {r.range_from}–{r.range_to ?? "∞"}</td>
                <td className="whitespace-nowrap px-3 py-2 text-(--text-muted)">{r.current_number}</td>
                <td className="whitespace-nowrap px-3 py-2 text-(--text-muted)">{r.valid_to}</td>
                <td className="whitespace-nowrap px-3 py-2 text-(--text-muted)">{r.environment === "1" ? "Producción" : "Habilitación"}</td>
                <td className="whitespace-nowrap px-3 py-2">
                  <span className={`rounded px-2 py-0.5 font-medium ${isExpiringSoon(r) ? "bg-(--color-warning-bg) text-(--color-warning-text)" : STATUS_CLASSES[r.status]}`}>
                    {isExpiringSoon(r) ? "Por vencer" : STATUS_LABELS[r.status]}
                  </span>
                </td>
                <td className="px-3 py-2 text-right">
                  <div className="flex items-center justify-end gap-1">
                    {r.status === "active" && (
                      <button type="button" title="Desactivar" onClick={() => onDeactivate(r)}
                        className="rounded p-1 text-(--color-danger) transition-colors hover:bg-(--bg-hover)">
                        <Ban className="h-3.5 w-3.5" />
                      </button>
                    )}
                    {(r.status === "expired" || r.status === "exhausted") && (
                      <button type="button" title="Archivar" onClick={() => onArchive(r)}
                        className="rounded p-1 text-(--text-muted) transition-colors hover:bg-(--bg-hover)">
                        <Archive className="h-3.5 w-3.5" />
                      </button>
                    )}
                    {r.status === "inactive" && wouldBeUsable(r) && (
                      <button type="button" title="Reactivar" onClick={() => onActivate(r)}
                        className="rounded p-1 text-(--color-success) transition-colors hover:bg-(--bg-hover)">
                        <RotateCcw className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Pagination
        offset={offset}
        count={page.length}
        hasNext={hasNext}
        onPrev={() => setOffset((o) => Math.max(0, o - RANGES_PAGE_SIZE))}
        onNext={() => setOffset((o) => o + RANGES_PAGE_SIZE)}
      />
    </>
  );
}

// ── Modal de importación desde DIAN ──────────────────────────────────────────────────────────

interface ImportModalProps {
  dianRanges: DianRange[];
  existingPrefixes: Set<string>;
  docTypes: { code: string; name: string }[];
  environment: string;
  onImport: (r: DianRange, docTypeCode: string, testSetId: string) => Promise<void>;
  onClose: () => void;
}

function ImportModal({ dianRanges, existingPrefixes, docTypes, environment, onImport, onClose }: ImportModalProps) {
  const [docTypeCodes, setDocTypeCodes] = useState<Record<number, string>>(() =>
    Object.fromEntries(dianRanges.map((r, i) => [i, r.suggested_doc_type_code ?? "01"]))
  );
  const [testSetIds, setTestSetIds] = useState<Record<number, string>>(() =>
    Object.fromEntries(dianRanges.map((_, i) => [i, ""]))
  );
  const [importingIdx, setImportingIdx] = useState<number | null>(null);
  const [importedIndices, setImportedIndices] = useState<Set<number>>(new Set());

  async function handleImport(r: DianRange, idx: number) {
    setImportingIdx(idx);
    try {
      await onImport(r, docTypeCodes[idx] ?? "01", testSetIds[idx] ?? "");
      setImportedIndices((prev) => new Set([...prev, idx]));
    } finally {
      setImportingIdx(null);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/50 p-4 pt-16">
      <div className="flex w-full max-w-2xl flex-col gap-4 rounded-lg bg-(--bg-primary) p-6 shadow-xl">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-semibold text-(--text-primary)">Sincronizar con la DIAN</h3>
            <p className="mt-0.5 text-xs text-(--text-muted)">
              Ambiente: {environment === "1" ? "Producción" : "Habilitación"} — selecciona el tipo de documento e importa.
            </p>
          </div>
          <button type="button" onClick={onClose} className="rounded p-1 text-(--text-muted) hover:bg-(--bg-hover) hover:text-(--text-primary)">
            <X className="h-4 w-4" />
          </button>
        </div>

        {dianRanges.length === 0 ? (
          <p className="text-xs text-(--text-secondary)">
            La DIAN no reportó rangos para este software en el ambiente {environment === "1" ? "de producción" : "de habilitación"}.
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            {dianRanges.map((r, i) => {
              const isImported = importedIndices.has(i);
              const alreadyExists = existingPrefixes.has(r.prefix);

              return (
                <div key={i} className="flex flex-col gap-3 rounded border border-(--border-color) p-4">
                  {/* Cabecera del rango */}
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-sm font-semibold text-(--text-primary)">{r.prefix}</span>
                      <span className="text-xs text-(--text-muted)">Res. {r.resolution_number}</span>
                    </div>
                    {isImported && (
                      <span className="text-xs font-medium text-(--color-success-text)">✓ Importado</span>
                    )}
                    {!isImported && alreadyExists && (
                      <span className="rounded bg-(--color-warning-bg) px-2 py-0.5 text-xs text-(--color-warning-text)">
                        Ya existe en el sistema
                      </span>
                    )}
                  </div>

                  {/* Datos del rango */}
                  <div className="grid grid-cols-2 gap-1 text-xs text-(--text-secondary)">
                    <span>Rango: <span className="font-mono text-(--text-primary)">{r.range_from} – {r.range_to}</span></span>
                    <span>Resolución: {r.resolution_date}</span>
                    <span>Vigente desde: {r.valid_from}</span>
                    <span>Vigente hasta: {r.valid_to}</span>
                  </div>

                  {/* Controles de importación (ocultos si ya fue importado) */}
                  {!isImported && (
                    <div className="grid grid-cols-12 gap-3 border-t border-(--border-light) pt-3">
                      <div className={environment === "2" ? "col-span-12 sm:col-span-6" : "col-span-12"}>
                        <Select
                          label="Tipo de documento"
                          required
                          value={docTypeCodes[i] ?? "01"}
                          onChange={(e) => setDocTypeCodes((prev) => ({ ...prev, [i]: e.target.value }))}
                        >
                          {docTypes.map((t) => (
                            <option key={t.code} value={t.code}>{t.name}</option>
                          ))}
                        </Select>
                      </div>
                      {environment === "2" && (
                        <div className="col-span-12 sm:col-span-6">
                          <Input
                            label="Set de pruebas"
                            value={testSetIds[i] ?? ""}
                            onChange={(e) => setTestSetIds((prev) => ({ ...prev, [i]: e.target.value }))}
                            placeholder="ID que asigna la DIAN"
                          />
                        </div>
                      )}
                      <div className="col-span-12 flex justify-end">
                        <Button
                          type="button"
                          loading={importingIdx === i}
                          disabled={importingIdx !== null}
                          onClick={() => handleImport(r, i)}
                        >
                          Importar rango
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}

        <div className="flex justify-end border-t border-(--border-light) pt-3">
          <Button type="button" variant="secondary" onClick={onClose}>Cerrar</Button>
        </div>
      </div>
    </div>
  );
}

// ── Panel principal ───────────────────────────────────────────────────────────────────────────

export function NumberingRangesPanel() {
  const { activeCompany } = useAuth();
  const [ranges, setRanges] = useState<NumberingRange[] | null>(null);
  const { data: docTypes } = useCatalog(listDianDocumentTypes);
  const [showForm, setShowForm] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const confirm = useConfirm();
  const toast = useToast();

  // Estado del modal de importación desde DIAN
  const [showImportModal, setShowImportModal] = useState(false);
  const [dianRanges, setDianRanges] = useState<DianRange[] | null>(null);
  const [importFetching, setImportFetching] = useState(false);

  function refresh() {
    listNumberingRanges()
      .then(setRanges)
      .catch((err) => setLoadError(err instanceof ApiError ? err.message : "No se pudieron cargar los rangos de numeración"));
  }

  useEffect(() => {
    refresh();
  }, []);

  function docTypeName(code: string): string {
    return docTypes.find((t) => t.code === code)?.name ?? code;
  }

  async function handleOpenImport() {
    setImportFetching(true);
    try {
      const data = await getDianNumberingRanges();
      setDianRanges(data);
      setShowImportModal(true);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo consultar la DIAN");
    } finally {
      setImportFetching(false);
    }
  }

  async function handleImportRange(r: DianRange, docTypeCode: string, testSetId: string) {
    const payload: CreateNumberingRangePayload = {
      dian_document_type_code: docTypeCode,
      prefix: r.prefix,
      resolution_number: r.resolution_number,
      resolution_date: r.resolution_date,
      range_from: r.range_from,
      range_to: r.range_to,
      valid_from: r.valid_from,
      valid_to: r.valid_to,
      environment: activeCompany?.environment ?? "2",
      technical_key: docTypeCode === "01" ? r.technical_key : undefined,
      test_set_id: activeCompany?.environment === "2" ? (testSetId || undefined) : undefined,
    };
    await createNumberingRange(payload);
    toast.success(`Rango "${r.prefix}" importado correctamente.`);
    refresh();
  }

  async function handleCreate(payload: CreateNumberingRangePayload) {
    setLoading(true);
    try {
      await createNumberingRange(payload);
      setShowForm(false);
      toast.success("Rango de numeración registrado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo registrar el rango de numeración");
    } finally {
      setLoading(false);
    }
  }

  async function handleDeactivate(r: NumberingRange) {
    if (!(await confirm(`¿Desactivar el rango "${r.prefix}"? Dejará de usarse para emitir facturas — podrás reactivarlo después si sigue siendo válido.`, { tone: "danger" }))) return;
    try {
      await deactivateNumberingRange(r.id);
      toast.success("Rango desactivado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo desactivar el rango");
    }
  }

  async function handleArchive(r: NumberingRange) {
    if (!(await confirm(`¿Archivar el rango "${r.prefix}"? Quedará oculto en esta vista. No se puede reactivar porque ya está ${r.status === "expired" ? "vencido" : "agotado"}.`))) return;
    try {
      await deactivateNumberingRange(r.id);
      toast.success("Rango archivado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo archivar el rango");
    }
  }

  async function handleActivate(r: NumberingRange) {
    if (!(await confirm(`¿Reactivar el rango "${r.prefix}"? Volverá a estar disponible para emitir facturas.`))) return;
    try {
      await activateNumberingRange(r.id);
      toast.success("Rango reactivado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo reactivar el rango");
    }
  }

  const [showInactive, setShowInactive] = useState(false);

  const activeRanges = ranges?.filter((r) => r.status === "active") ?? [];
  const inactiveRanges = ranges?.filter((r) => r.status !== "active") ?? [];
  const existingPrefixes = new Set(ranges?.map((r) => r.prefix) ?? []);

  return (
    <>
      <Card className="flex flex-col gap-3 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-xs font-semibold text-(--text-primary)">Rangos de numeración</h2>
          {ranges !== null && (
            <div className="flex items-center gap-2">
              {!showForm && (
                <Button
                  type="button"
                  variant="secondary"
                  icon={<Download className="h-3.5 w-3.5" />}
                  loading={importFetching}
                  onClick={handleOpenImport}
                >
                  Sincronizar con DIAN
                </Button>
              )}
              {!showForm && (
                <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowForm(true)}>
                  Nuevo rango
                </Button>
              )}
            </div>
          )}
        </div>

        {loadError && <Banner tone="danger">{loadError}</Banner>}

        {ranges === null && (
          <div className="flex min-h-20 items-center justify-center">
            <Spinner className="h-5 w-5 text-(--text-muted)" />
          </div>
        )}

        {showForm && (
          <NumberingRangeForm onSubmit={handleCreate} onCancel={() => setShowForm(false)} loading={loading} />
        )}

        {ranges?.length === 0 && !showForm && (
          <p className="text-xs text-(--text-secondary)">Todavía no hay rangos de numeración registrados.</p>
        )}

        {activeRanges.length > 0 && (
          <div className="flex flex-col gap-1.5">
            <p className="text-xs font-medium text-(--text-secondary)">Activos</p>
            <RangesTable rows={activeRanges} docTypeName={docTypeName} onDeactivate={handleDeactivate} onArchive={handleArchive} onActivate={handleActivate} />
          </div>
        )}

        {inactiveRanges.length > 0 && <div className="border-t border-(--border-light)" />}

        {inactiveRanges.length > 0 && (
          <div className="flex flex-col gap-1.5">
            <button
              type="button"
              onClick={() => setShowInactive((v) => !v)}
              className="flex w-fit items-center gap-1 text-xs font-medium text-(--text-muted) transition-colors hover:text-(--text-secondary)"
            >
              {showInactive ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
              Inactivos y vencidos ({inactiveRanges.length})
            </button>
            {showInactive && (
              <RangesTable rows={inactiveRanges} docTypeName={docTypeName} onDeactivate={handleDeactivate} onArchive={handleArchive} onActivate={handleActivate} />
            )}
          </div>
        )}
      </Card>

      {showImportModal && dianRanges !== null && (
        <ImportModal
          dianRanges={dianRanges}
          existingPrefixes={existingPrefixes}
          docTypes={docTypes}
          environment={activeCompany?.environment ?? "2"}
          onImport={handleImportRange}
          onClose={() => setShowImportModal(false)}
        />
      )}
    </>
  );
}
