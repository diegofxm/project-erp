import { useEffect, useState } from "react";
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import {
  TrendingUp,
  FileText,
  CheckCircle2,
  Clock,
  BarChart2,
  ArrowRight,
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

// ── Helpers ──────────────────────────────────────────────────────────────────

const MONTHS_ES = ["Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic"];

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
  "01": "#3498db",
  "91": "#f39c12",
  "92": "#27ae60",
  "05": "#9b59b6",
};

const DOC_TYPE_SHORT: Record<string, string> = {
  "01": "FE",
  "91": "NC",
  "92": "ND",
  "05": "DS",
};

function docTypeLabel(code: string): string {
  const map: Record<string, string> = {
    "01": "Factura Electrónica",
    "91": "Nota Crédito",
    "92": "Nota Débito",
    "05": "Doc. Soporte",
  };
  return map[code] ?? code;
}

function routeForType(code: string): string {
  const routes: Record<string, string> = {
    "01": "/documents/invoices",
    "91": "/documents/credit-notes",
    "92": "/documents/debit-notes",
    "05": "/documents/support-documents",
  };
  return routes[code] ?? "/documents/invoices";
}

// ── Componentes internos ─────────────────────────────────────────────────────

interface KpiCardProps {
  title: string;
  value: string;
  icon: React.ComponentType<{ className?: string }>;
  delta?: { label: string; positive: boolean } | null;
  subtitle?: string;
}

function KpiCard({ title, value, icon: Icon, delta, subtitle }: KpiCardProps) {
  return (
    <Card className="p-5 flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">{title}</span>
        <Icon className="h-4 w-4 text-(--accent-primary)" />
      </div>
      <div className="text-2xl font-bold tabular-nums text-(--text-primary)">{value}</div>
      <div className="flex items-center gap-2 text-xs min-h-[1rem]">
        {delta != null && (
          <span className={delta.positive ? "text-(--color-success)" : "text-(--color-danger)"}>
            {delta.positive ? "↑" : "↓"} {delta.label}
          </span>
        )}
        {subtitle && <span className="text-(--text-muted)">{subtitle}</span>}
      </div>
    </Card>
  );
}

function RevenueTooltip({ active, payload }: { active?: boolean; payload?: { value: number; payload: { count: number } }[] }) {
  if (!active || !payload?.length) return null;
  const revenue = payload[0].value;
  const count = payload[0].payload.count;
  return (
    <div className="rounded-md border border-(--border-color) bg-(--bg-secondary) px-3 py-2 shadow-md text-xs">
      <p className="text-(--text-secondary) mb-0.5">
        Ingresos:{" "}
        <span className="font-semibold text-(--text-primary)">{formatCOP.format(revenue / 100)}</span>
      </p>
      <p className="text-(--text-secondary)">
        Documentos: <span className="font-semibold text-(--text-primary)">{count}</span>
      </p>
    </div>
  );
}

function TypeTooltip({ active, payload }: { active?: boolean; payload?: { payload: BillingStats["by_type"][0] }[] }) {
  if (!active || !payload?.length) return null;
  const d = payload[0].payload;
  return (
    <div className="rounded-md border border-(--border-color) bg-(--bg-secondary) px-3 py-2 shadow-md text-xs">
      <p className="font-semibold text-(--text-primary) mb-0.5">{d.type_name}</p>
      <p className="text-(--text-secondary)">
        Documentos: <span className="font-semibold text-(--text-primary)">{d.count}</span>
      </p>
      {d.revenue_cents > 0 && (
        <p className="text-(--text-secondary)">
          Total: <span className="font-semibold text-(--text-primary)">{formatCOP.format(d.revenue_cents / 100)}</span>
        </p>
      )}
    </div>
  );
}

// ── Página principal ──────────────────────────────────────────────────────────

export function DashboardPage() {
  const { activeIssuer } = useAuth();
  const [stats, setStats] = useState<BillingStats | null>(null);
  const [recentDocs, setRecentDocs] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!activeIssuer) return;
    let cancelled = false;
    setLoading(true);
    setError(null);

    Promise.all([getBillingStats(), listDocuments({ limit: 6 })])
      .then(([statsData, docs]) => {
        if (cancelled) return;
        setStats(statsData);
        setRecentDocs(docs);
      })
      .catch((err: Error) => {
        if (cancelled) return;
        setError(err.message ?? "No se pudieron cargar las métricas");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [activeIssuer?.id]);

  const cm = stats?.current_month;
  const pm = stats?.previous_month;

  const acceptanceRateCurrent = cm && cm.document_count > 0 ? (cm.accepted_count / cm.document_count) * 100 : 0;
  const acceptanceRatePrev = pm && pm.document_count > 0 ? (pm.accepted_count / pm.document_count) * 100 : 0;

  const seriesData = (stats?.series ?? []).map((s) => ({ ...s, label: monthLabel(s.month) }));

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4">
        <Card className="p-4 text-sm text-(--color-danger-text) bg-(--color-danger-bg)">
          {error}
        </Card>
      </div>
    );
  }

  return (
    <div className="p-4 flex flex-col gap-6">
      {/* Encabezado */}
      <div className="flex items-center gap-2">
        <BarChart2 className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        <h1 className="text-sm font-semibold text-(--text-primary)">Panel de Control</h1>
        {activeIssuer && (
          <span className="ml-auto text-xs text-(--text-muted)">
            {activeIssuer.business_name}
            {" · "}
            <span className={activeIssuer.environment === "1" ? "text-(--color-success)" : "text-(--color-warning-text)"}>
              {activeIssuer.environment === "1" ? "Producción" : "Habilitación"}
            </span>
          </span>
        )}
      </div>

      {/* KPIs */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard
          title="Ingresos del mes"
          value={cm ? formatCOP.format(cm.revenue_cents / 100) : "—"}
          icon={TrendingUp}
          delta={cm && pm ? pctDelta(cm.revenue_cents, pm.revenue_cents) : null}
          subtitle="vs. mes anterior"
        />
        <KpiCard
          title="Documentos emitidos"
          value={cm ? String(cm.document_count) : "—"}
          icon={FileText}
          delta={cm && pm ? pctDelta(cm.document_count, pm.document_count) : null}
          subtitle="vs. mes anterior"
        />
        <KpiCard
          title="Tasa de aprobación"
          value={cm && cm.document_count > 0 ? `${Math.round(acceptanceRateCurrent)}%` : "—"}
          icon={CheckCircle2}
          delta={cm && pm && pm.document_count > 0 ? pctDelta(acceptanceRateCurrent, acceptanceRatePrev) : null}
          subtitle="aceptados / emitidos"
        />
        <KpiCard
          title="Borradores pendientes"
          value={cm ? String(cm.draft_count) : "—"}
          icon={Clock}
          subtitle="sin confirmar"
        />
      </div>

      {/* Gráficas */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Tendencia de ingresos — 2/3 del ancho */}
        <Card className="lg:col-span-2 p-4 flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
              Tendencia de ingresos
            </h2>
            <span className="text-xs text-(--text-muted)">Últimos 12 meses</span>
          </div>

          {seriesData.length === 0 ? (
            <div className="flex items-center justify-center h-48 text-sm text-(--text-muted)">
              Sin datos aún
            </div>
          ) : (
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={seriesData} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
                <defs>
                  <linearGradient id="revenueGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="15%" stopColor="#3498db" stopOpacity={0.25} />
                    <stop offset="95%" stopColor="#3498db" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#d0d0d0" vertical={false} />
                <XAxis
                  dataKey="label"
                  tick={{ fontSize: 11, fill: "#5a6c7d" }}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tickFormatter={formatRevenueTick}
                  tick={{ fontSize: 11, fill: "#5a6c7d" }}
                  axisLine={false}
                  tickLine={false}
                  width={56}
                />
                <Tooltip content={<RevenueTooltip />} cursor={{ stroke: "#3498db", strokeWidth: 1, strokeDasharray: "4 2" }} />
                <Area
                  type="monotone"
                  dataKey="revenue_cents"
                  stroke="#3498db"
                  strokeWidth={2}
                  fill="url(#revenueGradient)"
                  dot={false}
                  activeDot={{ r: 4, fill: "#3498db", stroke: "#fff", strokeWidth: 2 }}
                />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </Card>

        {/* Documentos por tipo — 1/3 del ancho */}
        <Card className="p-4 flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
              Por tipo
            </h2>
            <span className="text-xs text-(--text-muted)">Este mes</span>
          </div>

          {!stats?.by_type.length ? (
            <div className="flex items-center justify-center h-48 text-sm text-(--text-muted)">
              Sin datos aún
            </div>
          ) : (
            <>
              <ResponsiveContainer width="100%" height={160}>
                <BarChart
                  data={stats.by_type}
                  layout="vertical"
                  margin={{ top: 0, right: 8, left: 0, bottom: 0 }}
                >
                  <CartesianGrid strokeDasharray="3 3" stroke="#d0d0d0" horizontal={false} />
                  <XAxis type="number" tick={{ fontSize: 11, fill: "#5a6c7d" }} axisLine={false} tickLine={false} />
                  <YAxis
                    type="category"
                    dataKey="type_code"
                    tickFormatter={(code: string) => DOC_TYPE_SHORT[code] ?? code}
                    tick={{ fontSize: 11, fill: "#5a6c7d" }}
                    axisLine={false}
                    tickLine={false}
                    width={28}
                  />
                  <Tooltip content={<TypeTooltip />} cursor={{ fill: "rgba(0,0,0,0.04)" }} />
                  <Bar dataKey="count" radius={[0, 3, 3, 0]}>
                    {stats.by_type.map((entry) => (
                      <Cell key={entry.type_code} fill={TYPE_COLORS[entry.type_code] ?? "#3498db"} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>

              {/* Leyenda manual */}
              <div className="flex flex-col gap-1.5 mt-1">
                {stats.by_type.map((t) => (
                  <div key={t.type_code} className="flex items-center justify-between text-xs">
                    <div className="flex items-center gap-1.5">
                      <span
                        className="inline-block h-2 w-2 rounded-sm shrink-0"
                        style={{ backgroundColor: TYPE_COLORS[t.type_code] ?? "#3498db" }}
                      />
                      <span className="text-(--text-secondary)">{docTypeLabel(t.type_code)}</span>
                    </div>
                    <span className="font-medium tabular-nums text-(--text-primary)">{t.count}</span>
                  </div>
                ))}
              </div>
            </>
          )}
        </Card>
      </div>

      {/* Acumulado del año */}
      {stats?.ytd && stats.ytd.document_count > 0 && (
        <Card className="p-4">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted) mb-3">
            Acumulado del año
          </h2>
          <div className="grid grid-cols-3 gap-4 text-center">
            <div>
              <div className="text-xl font-bold tabular-nums text-(--text-primary)">
                {formatCOP.format(stats.ytd.revenue_cents / 100)}
              </div>
              <div className="text-xs text-(--text-muted) mt-0.5">Ingresos totales</div>
            </div>
            <div>
              <div className="text-xl font-bold tabular-nums text-(--text-primary)">{stats.ytd.document_count}</div>
              <div className="text-xs text-(--text-muted) mt-0.5">Documentos emitidos</div>
            </div>
            <div>
              <div className="text-xl font-bold tabular-nums text-(--text-primary)">
                {stats.ytd.document_count > 0 ? `${Math.round((stats.ytd.accepted_count / stats.ytd.document_count) * 100)}%` : "—"}
              </div>
              <div className="text-xs text-(--text-muted) mt-0.5">Tasa de aprobación</div>
            </div>
          </div>
        </Card>
      )}

      {/* Actividad reciente */}
      <Card className="p-4 flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
            Actividad reciente
          </h2>
          <Link
            to="/documents/invoices"
            className="flex items-center gap-1 text-xs text-(--accent-primary) hover:underline"
          >
            Ver todos <ArrowRight className="h-3 w-3" />
          </Link>
        </div>

        {recentDocs.length === 0 ? (
          <p className="text-sm text-(--text-muted) py-4 text-center">Aún no hay documentos emitidos</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-(--border-light)">
                  <th className="text-left py-2 pr-3 font-medium text-(--text-muted)">Número</th>
                  <th className="text-left py-2 pr-3 font-medium text-(--text-muted)">Tipo</th>
                  <th className="text-left py-2 pr-3 font-medium text-(--text-muted)">Tercero</th>
                  <th className="text-right py-2 pr-3 font-medium text-(--text-muted)">Total</th>
                  <th className="text-left py-2 font-medium text-(--text-muted)">Estado</th>
                </tr>
              </thead>
              <tbody>
                {recentDocs.map((doc, i) => {
                  const counterpart = doc.dian_document_type_code === "05" ? doc.vendor : doc.customer;
                  return (
                    <tr
                      key={doc.id}
                      className={`border-b border-(--border-light) ${i % 2 === 0 ? "bg-(--bg-primary)" : "bg-(--bg-secondary)"}`}
                    >
                      <td className="py-2 pr-3 font-mono text-(--text-primary)">
                        <Link
                          to={`${routeForType(doc.dian_document_type_code)}/${doc.id}`}
                          className="hover:text-(--accent-primary) hover:underline"
                        >
                          {doc.prefix || "—"}
                          {doc.number ?? ""}
                        </Link>
                      </td>
                      <td className="py-2 pr-3">
                        <span
                          className="px-1.5 py-0.5 rounded text-[10px] font-semibold text-white"
                          style={{ backgroundColor: TYPE_COLORS[doc.dian_document_type_code] ?? "#5a6c7d" }}
                        >
                          {DOC_TYPE_SHORT[doc.dian_document_type_code] ?? doc.dian_document_type_code}
                        </span>
                      </td>
                      <td className="py-2 pr-3 text-(--text-secondary) max-w-[180px] truncate">
                        {counterpart?.name ?? "—"}
                      </td>
                      <td className="py-2 pr-3 text-right font-medium tabular-nums text-(--text-primary)">
                        {doc.totals ? formatCOP.format(doc.totals.payable_cents / 100) : "—"}
                      </td>
                      <td className="py-2">
                        <StatusBadge status={doc.status as DocumentStatus} />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
