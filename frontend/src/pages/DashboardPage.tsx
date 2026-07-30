import { useEffect, useRef, useState } from "react";
import { ReactGridLayout, WidthProvider } from "react-grid-layout/legacy";
import {
  AreaChart, Area, BarChart, Bar, XAxis, YAxis,
  CartesianGrid, Tooltip, ResponsiveContainer, Cell,
} from "recharts";
import {
  TrendingUp, FileText, CheckCircle2, Clock,
  LayoutDashboard, ArrowRight, Settings2, Plus, RotateCcw, X, GripHorizontal,
} from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { StatusBadge } from "../components/invoice-form/StatusBadge";
import { getBillingStats, type BillingStats } from "../lib/stats";
import { listDocuments } from "../lib/documents";
import type { Document, DocumentStatus } from "../lib/types";
import { formatCOP } from "../lib/currency";
import { Link } from "react-router";
import {
  useDashboardLayout,
  WIDGET_LABELS,
  type WidgetId,
} from "../hooks/useDashboardLayout";

const GridLayout = WidthProvider(ReactGridLayout);

// ── Helpers ──────────────────────────────────────────────────────────────────

const MONTHS_ES = ["Ene","Feb","Mar","Abr","May","Jun","Jul","Ago","Sep","Oct","Nov","Dic"];

function monthLabel(ym: string): string {
  const [year, month] = ym.split("-");
  return `${MONTHS_ES[parseInt(month, 10) - 1]} '${year.slice(2)}`;
}

function pctDelta(current: number, prev: number): { label: string; positive: boolean } | null {
  if (prev === 0 && current === 0) return null;
  if (prev === 0) return { label: "nuevo", positive: true };
  const pct = Math.round(((current - prev) / prev) * 100);
  return { label: `${pct > 0 ? "+" : ""}${pct}%`, positive: pct >= 0 };
}

function formatRevenueTick(cents: number): string {
  const v = cents / 100;
  if (v >= 1_000_000) return `$${(v / 1_000_000).toFixed(1)}M`;
  if (v >= 1_000) return `$${(v / 1_000).toFixed(0)}k`;
  return `$${v.toFixed(0)}`;
}

const TYPE_COLORS: Record<string, string> = {
  "01": "#3498db", "91": "#f39c12", "92": "#27ae60", "05": "#9b59b6",
};
const DOC_TYPE_SHORT: Record<string, string> = {
  "01": "FE", "91": "NC", "92": "ND", "05": "DS",
};
function docTypeLabel(code: string): string {
  return ({ "01": "Factura Electrónica", "91": "Nota Crédito", "92": "Nota Débito", "05": "Doc. Soporte" } as Record<string,string>)[code] ?? code;
}
function routeForType(code: string): string {
  return ({ "01": "/documents/invoices", "91": "/documents/credit-notes", "92": "/documents/debit-notes", "05": "/documents/support-documents" } as Record<string,string>)[code] ?? "/documents/invoices";
}

// ── AddWidgetDropdown ────────────────────────────────────────────────────────

function AddWidgetDropdown({
  hidden, onShow, onReset,
}: {
  hidden: WidgetId[];
  onShow: (id: WidgetId) => void;
  onReset: () => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function fn(e: PointerEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("pointerdown", fn);
    return () => document.removeEventListener("pointerdown", fn);
  }, [open]);

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-1.5 rounded border border-(--border-color) bg-(--bg-primary) px-2.5 py-1 text-xs text-(--text-secondary) hover:bg-(--bg-secondary)"
      >
        <Plus className="h-3.5 w-3.5" />
        Agregar
      </button>

      {open && (
        <div className="absolute right-0 top-full z-50 mt-1 w-52 rounded-lg border border-(--border-color) bg-(--bg-primary) py-1 shadow-lg">
          {hidden.length === 0 ? (
            <p className="px-3 py-2 text-xs text-(--text-muted)">Todos los widgets están visibles</p>
          ) : (
            <>
              <p className="px-3 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-wider text-(--text-muted)">
                Ocultos
              </p>
              {hidden.map((id) => (
                <button
                  key={id}
                  type="button"
                  onClick={() => { onShow(id); setOpen(false); }}
                  className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs text-(--text-secondary) hover:bg-(--bg-secondary)"
                >
                  <Plus className="h-3 w-3 shrink-0 text-(--accent-primary)" />
                  {WIDGET_LABELS[id]}
                </button>
              ))}
            </>
          )}
          <div className="mt-1 border-t border-(--border-color) pt-1">
            <button
              type="button"
              onClick={() => { onReset(); setOpen(false); }}
              className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs text-(--text-muted) hover:bg-(--bg-secondary)"
            >
              <RotateCcw className="h-3 w-3" />
              Restaurar por defecto
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Tooltips ─────────────────────────────────────────────────────────────────

function RevenueTooltip({ active, payload }: { active?: boolean; payload?: { value: number; payload: { count: number } }[] }) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-md border border-(--border-color) bg-(--bg-secondary) px-3 py-2 shadow-md text-xs">
      <p className="mb-0.5 text-(--text-secondary)">
        Ingresos: <span className="font-semibold text-(--text-primary)">{formatCOP.format(payload[0].value / 100)}</span>
      </p>
      <p className="text-(--text-secondary)">
        Documentos: <span className="font-semibold text-(--text-primary)">{payload[0].payload.count}</span>
      </p>
    </div>
  );
}

function TypeTooltip({ active, payload }: { active?: boolean; payload?: { payload: BillingStats["by_type"][0] }[] }) {
  if (!active || !payload?.length) return null;
  const d = payload[0].payload;
  return (
    <div className="rounded-md border border-(--border-color) bg-(--bg-secondary) px-3 py-2 shadow-md text-xs">
      <p className="mb-0.5 font-semibold text-(--text-primary)">{d.type_name}</p>
      <p className="text-(--text-secondary)">Documentos: <span className="font-semibold text-(--text-primary)">{d.count}</span></p>
      {d.revenue_cents > 0 && (
        <p className="text-(--text-secondary)">Total: <span className="font-semibold text-(--text-primary)">{formatCOP.format(d.revenue_cents / 100)}</span></p>
      )}
    </div>
  );
}

// ── Página principal ──────────────────────────────────────────────────────────

export function DashboardPage() {
  const { activeCompany } = useAuth();
  const [stats, setStats] = useState<BillingStats | null>(null);
  const [recentDocs, setRecentDocs] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editMode, setEditMode] = useState(false);

  const { visibleItems, hidden, updateItems, hide, show, reset } = useDashboardLayout();

  useEffect(() => {
    if (!activeCompany) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    Promise.all([getBillingStats(), listDocuments({ limit: 6 })])
      .then(([s, d]) => { if (!cancelled) { setStats(s); setRecentDocs(d); } })
      .catch((err: Error) => { if (!cancelled) setError(err.message ?? "No se pudieron cargar las métricas"); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [activeCompany?.id]);

  const cm = stats?.current_month;
  const pm = stats?.previous_month;
  const acceptRate = cm && cm.document_count > 0 ? (cm.accepted_count / cm.document_count) * 100 : 0;
  const acceptRatePrev = pm && pm.document_count > 0 ? (pm.accepted_count / pm.document_count) * 100 : 0;
  const seriesData = (stats?.series ?? []).map((s) => ({ ...s, label: monthLabel(s.month) }));

  // w_recent se renderiza fuera del grid (altura automática según contenido)
  // w_ytd se oculta si no hay datos y no estamos en edición
  const gridItems = visibleItems.filter((item) => {
    if (item.i === "w_recent") return false;
    if (item.i === "w_ytd") {
      const hasData = stats?.ytd && stats.ytd.document_count > 0;
      return hasData || editMode;
    }
    return true;
  });
  const showRecent = !hidden.includes("w_recent");

  // ── Renderizador por widget ───────────────────────────────────────────────

  function renderWidget(id: WidgetId): React.ReactNode {
    switch (id) {
      case "w_revenue":
        return (
          <Card className="flex h-full flex-col gap-3 p-5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Ingresos del mes</span>
              <TrendingUp className="h-4 w-4 text-(--accent-primary)" />
            </div>
            <div className="text-2xl font-bold tabular-nums text-(--text-primary)">
              {cm ? formatCOP.format(cm.revenue_cents / 100) : "—"}
            </div>
            <KpiFooter delta={cm && pm ? pctDelta(cm.revenue_cents, pm.revenue_cents) : null} subtitle="vs. mes anterior" />
          </Card>
        );
      case "w_docs":
        return (
          <Card className="flex h-full flex-col gap-3 p-5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Documentos emitidos</span>
              <FileText className="h-4 w-4 text-(--accent-primary)" />
            </div>
            <div className="text-2xl font-bold tabular-nums text-(--text-primary)">{cm ? String(cm.document_count) : "—"}</div>
            <KpiFooter delta={cm && pm ? pctDelta(cm.document_count, pm.document_count) : null} subtitle="vs. mes anterior" />
          </Card>
        );
      case "w_acceptance":
        return (
          <Card className="flex h-full flex-col gap-3 p-5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Tasa de aprobación</span>
              <CheckCircle2 className="h-4 w-4 text-(--accent-primary)" />
            </div>
            <div className="text-2xl font-bold tabular-nums text-(--text-primary)">
              {cm && cm.document_count > 0 ? `${Math.round(acceptRate)}%` : "—"}
            </div>
            <KpiFooter delta={cm && pm && pm.document_count > 0 ? pctDelta(acceptRate, acceptRatePrev) : null} subtitle="aceptados / emitidos" />
          </Card>
        );
      case "w_drafts":
        return (
          <Card className="flex h-full flex-col gap-3 p-5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Borradores pendientes</span>
              <Clock className="h-4 w-4 text-(--accent-primary)" />
            </div>
            <div className="text-2xl font-bold tabular-nums text-(--text-primary)">{cm ? String(cm.draft_count) : "—"}</div>
            <KpiFooter subtitle="sin confirmar" />
          </Card>
        );
      case "w_chart_area":
        return (
          <Card className="flex h-full flex-col gap-3 p-4">
            <div className="flex items-center justify-between">
              <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Tendencia de ingresos</h2>
              <span className="text-xs text-(--text-muted)">Últimos 12 meses</span>
            </div>
            {seriesData.length === 0 ? (
              <Empty />
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={seriesData} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
                  <defs>
                    <linearGradient id="revenueGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="15%" stopColor="#3498db" stopOpacity={0.25} />
                      <stop offset="95%" stopColor="#3498db" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#d0d0d0" vertical={false} />
                  <XAxis dataKey="label" tick={{ fontSize: 11, fill: "#5a6c7d" }} axisLine={false} tickLine={false} />
                  <YAxis tickFormatter={formatRevenueTick} tick={{ fontSize: 11, fill: "#5a6c7d" }} axisLine={false} tickLine={false} width={56} />
                  <Tooltip content={<RevenueTooltip />} cursor={{ stroke: "#3498db", strokeWidth: 1, strokeDasharray: "4 2" }} />
                  <Area type="monotone" dataKey="revenue_cents" stroke="#3498db" strokeWidth={2} fill="url(#revenueGradient)" dot={false} activeDot={{ r: 4, fill: "#3498db", stroke: "#fff", strokeWidth: 2 }} />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </Card>
        );
      case "w_chart_type":
        return (
          <Card className="flex h-full flex-col gap-3 p-4">
            <div className="flex items-center justify-between">
              <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Por tipo</h2>
              <span className="text-xs text-(--text-muted)">Este mes</span>
            </div>
            {!stats?.by_type.length ? <Empty /> : (
              <>
                <ResponsiveContainer width="100%" height={160}>
                  <BarChart data={stats.by_type} layout="vertical" margin={{ top: 0, right: 8, left: 0, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#d0d0d0" horizontal={false} />
                    <XAxis type="number" tick={{ fontSize: 11, fill: "#5a6c7d" }} axisLine={false} tickLine={false} />
                    <YAxis type="category" dataKey="type_code" tickFormatter={(c: string) => DOC_TYPE_SHORT[c] ?? c} tick={{ fontSize: 11, fill: "#5a6c7d" }} axisLine={false} tickLine={false} width={28} />
                    <Tooltip content={<TypeTooltip />} cursor={{ fill: "rgba(0,0,0,0.04)" }} />
                    <Bar dataKey="count" radius={[0, 3, 3, 0]}>
                      {stats.by_type.map((e) => <Cell key={e.type_code} fill={TYPE_COLORS[e.type_code] ?? "#3498db"} />)}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
                <div className="mt-1 flex flex-col gap-1.5">
                  {stats.by_type.map((t) => (
                    <div key={t.type_code} className="flex items-center justify-between text-xs">
                      <div className="flex items-center gap-1.5">
                        <span className="inline-block h-2 w-2 shrink-0 rounded-sm" style={{ backgroundColor: TYPE_COLORS[t.type_code] ?? "#3498db" }} />
                        <span className="text-(--text-secondary)">{docTypeLabel(t.type_code)}</span>
                      </div>
                      <span className="font-medium tabular-nums text-(--text-primary)">{t.count}</span>
                    </div>
                  ))}
                </div>
              </>
            )}
          </Card>
        );
      case "w_ytd": {
        const hasData = stats?.ytd && stats.ytd.document_count > 0;
        return (
          <Card className="flex h-full flex-col justify-center p-4">
            <h2 className="mb-3 text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Acumulado del año</h2>
            {!hasData ? <p className="text-xs text-(--text-muted)">Sin datos aún</p> : (
              <div className="grid grid-cols-3 gap-4 text-center">
                <div>
                  <div className="text-xl font-bold tabular-nums text-(--text-primary)">{formatCOP.format(stats!.ytd.revenue_cents / 100)}</div>
                  <div className="mt-0.5 text-xs text-(--text-muted)">Ingresos totales</div>
                </div>
                <div>
                  <div className="text-xl font-bold tabular-nums text-(--text-primary)">{stats!.ytd.document_count}</div>
                  <div className="mt-0.5 text-xs text-(--text-muted)">Documentos emitidos</div>
                </div>
                <div>
                  <div className="text-xl font-bold tabular-nums text-(--text-primary)">
                    {`${Math.round((stats!.ytd.accepted_count / stats!.ytd.document_count) * 100)}%`}
                  </div>
                  <div className="mt-0.5 text-xs text-(--text-muted)">Tasa de aprobación</div>
                </div>
              </div>
            )}
          </Card>
        );
      }
      case "w_recent":
        return (
          <Card className="flex flex-col gap-3 p-4">
            <div className="flex items-center justify-between">
              <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Actividad reciente</h2>
              <Link to="/documents/invoices" className="flex items-center gap-1 text-xs text-(--accent-primary) hover:underline">
                Ver todos <ArrowRight className="h-3 w-3" />
              </Link>
            </div>
            {recentDocs.length === 0 ? (
              <p className="py-4 text-center text-sm text-(--text-muted)">Aún no hay documentos emitidos</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-xs">
                  <thead>
                    <tr className="border-b border-(--border-light)">
                      <th className="py-2 pr-3 text-left font-medium text-(--text-muted)">Número</th>
                      <th className="py-2 pr-3 text-left font-medium text-(--text-muted)">Tipo</th>
                      <th className="py-2 pr-3 text-left font-medium text-(--text-muted)">Tercero</th>
                      <th className="py-2 pr-3 text-right font-medium text-(--text-muted)">Total</th>
                      <th className="py-2 text-left font-medium text-(--text-muted)">Estado</th>
                    </tr>
                  </thead>
                  <tbody>
                    {recentDocs.map((doc, i) => {
                      const cp = doc.dian_document_type_code === "05" ? doc.vendor : doc.customer;
                      return (
                        <tr key={doc.id} className={`border-b border-(--border-light) ${i % 2 === 0 ? "bg-(--bg-primary)" : "bg-(--bg-secondary)"}`}>
                          <td className="py-2 pr-3 font-mono text-(--text-primary)">
                            <Link to={`${routeForType(doc.dian_document_type_code)}/${doc.id}`} className="hover:text-(--accent-primary) hover:underline">
                              {doc.prefix || "—"}{doc.number ?? ""}
                            </Link>
                          </td>
                          <td className="py-2 pr-3">
                            <span className="rounded px-1.5 py-0.5 text-[10px] font-semibold text-white" style={{ backgroundColor: TYPE_COLORS[doc.dian_document_type_code] ?? "#5a6c7d" }}>
                              {DOC_TYPE_SHORT[doc.dian_document_type_code] ?? doc.dian_document_type_code}
                            </span>
                          </td>
                          <td className="max-w-45 truncate py-2 pr-3 text-(--text-secondary)">{cp?.name ?? "—"}</td>
                          <td className="py-2 pr-3 text-right font-medium tabular-nums text-(--text-primary)">
                            {doc.totals ? formatCOP.format(doc.totals.payable_cents / 100) : "—"}
                          </td>
                          <td className="py-2"><StatusBadge status={doc.status as DocumentStatus} /></td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        );
    }
  }

  // ── Loading / error ───────────────────────────────────────────────────────

  if (loading) return <div className="flex h-64 items-center justify-center"><Spinner /></div>;
  if (error) return <div className="p-4"><Card className="p-4 text-sm text-(--color-danger-text) bg-(--color-danger-bg)">{error}</Card></div>;

  return (
    <div className="flex flex-col gap-4 p-4">
      {/* Encabezado */}
      <div className="flex items-center gap-2">
        <LayoutDashboard className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        <h1 className="text-sm font-semibold text-(--text-primary)">Panel de Control</h1>
        {activeCompany && (
          <span className="mr-2 ml-auto text-xs text-(--text-muted)">
            {activeCompany.trade_name || activeCompany.business_name} ·{" "}
            <span className={activeCompany.environment === "1" ? "text-(--color-success)" : "text-(--color-warning-text)"}>
              {activeCompany.environment === "1" ? "Producción" : "Habilitación"}
            </span>
          </span>
        )}
        {!editMode && (
          <button
            type="button"
            onClick={() => setEditMode(true)}
            className="flex items-center gap-1.5 rounded border border-(--border-color) px-2.5 py-1 text-xs text-(--text-secondary) hover:bg-(--bg-secondary)"
          >
            <Settings2 className="h-3.5 w-3.5" />
            Personalizar
          </button>
        )}
      </div>

      {/* Barra de modo edición */}
      {editMode && (
        <div className="flex items-center gap-2 rounded-lg border border-dashed border-(--accent-primary) bg-(--color-info-bg) px-3 py-2">
          <Settings2 className="h-3.5 w-3.5 shrink-0 text-(--accent-primary)" />
          <span className="flex-1 text-xs font-medium text-(--accent-primary)">
            Toma la barra ≡ para mover · arrastra la esquina ↘ para redimensionar · ✕ para ocultar
          </span>
          <AddWidgetDropdown hidden={hidden} onShow={show} onReset={reset} />
          <button
            type="button"
            onClick={() => setEditMode(false)}
            className="rounded bg-(--accent-primary) px-3 py-1 text-xs font-medium text-white hover:opacity-90"
          >
            Listo
          </button>
        </div>
      )}

      {/* Grid de widgets (w_recent se renderiza aparte abajo) */}
      {gridItems.length === 0 && !showRecent ? (
        <div className="py-16 text-center">
          <p className="text-sm text-(--text-muted)">No hay widgets visibles.</p>
          <button type="button" onClick={() => setEditMode(true)} className="mt-2 text-xs text-(--accent-primary) hover:underline">
            Personalizar dashboard
          </button>
        </div>
      ) : gridItems.length > 0 && (
        <GridLayout
          layout={gridItems}
          cols={12}
          rowHeight={40}
          isDraggable={editMode}
          isResizable={editMode}
          onLayoutChange={updateItems}
          margin={[16, 16]}
          containerPadding={[0, 0]}
          resizeHandles={["se"]}
          draggableCancel=".react-resizable-handle"
        >
          {gridItems.map((item) => (
            // h-full es esencial: el .react-grid-item tiene height fijo por RGL,
            // el wrapper debe llenarlo para que el Card ocupe toda la celda
            <div key={item.i} className={`flex h-full flex-col ${editMode ? "cursor-grab active:cursor-grabbing" : ""}`}>
              {/* Barra de título en modo edición */}
              {editMode && (
                <div className="flex shrink-0 select-none items-center gap-2 rounded-t-lg border border-b-0 border-(--border-color) bg-(--bg-tertiary) px-2 py-1">
                  <GripHorizontal className="h-3.5 w-3.5 shrink-0 text-(--text-muted)" />
                  <span className="flex-1 truncate text-[10px] font-semibold uppercase tracking-wider text-(--text-muted)">
                    {WIDGET_LABELS[item.i]}
                  </span>
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); hide(item.i); }}
                    aria-label={`Ocultar ${WIDGET_LABELS[item.i]}`}
                    className="rounded p-0.5 text-(--text-muted) transition-colors hover:text-red-500"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              )}
              <div className="min-h-0 flex-1">
                {renderWidget(item.i)}
              </div>
            </div>
          ))}
        </GridLayout>
      )}

      {/* Actividad reciente — fuera del grid, altura automática según datos */}
      {showRecent && (
        <div>
          {editMode && (
            <div className="mb-1 flex items-center justify-end">
              <button
                type="button"
                onClick={() => hide("w_recent")}
                className="flex items-center gap-1 rounded border border-(--border-color) px-2 py-0.5 text-[10px] text-(--text-muted) hover:text-red-500"
              >
                <X className="h-3 w-3" /> Ocultar lista
              </button>
            </div>
          )}
          {renderWidget("w_recent")}
        </div>
      )}
    </div>
  );
}

// ── Subcomponentes pequeños ───────────────────────────────────────────────────

function KpiFooter({ delta, subtitle }: { delta?: { label: string; positive: boolean } | null; subtitle?: string }) {
  return (
    <div className="flex min-h-4 items-center gap-2 text-xs">
      {delta != null && (
        <span className={delta.positive ? "text-(--color-success)" : "text-(--color-danger)"}>
          {delta.positive ? "↑" : "↓"} {delta.label}
        </span>
      )}
      {subtitle && <span className="text-(--text-muted)">{subtitle}</span>}
    </div>
  );
}

function Empty() {
  return <div className="flex h-48 items-center justify-center text-sm text-(--text-muted)">Sin datos aún</div>;
}
