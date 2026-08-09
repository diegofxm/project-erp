import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { ArrowRight, CheckCircle2, Clock, FileText, TrendingUp } from "lucide-react";
import { listDocuments } from "../lib/documents";
import { computeBillingStats } from "../lib/electronicStats";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { addDaysColombiaISO, formatDateOnly } from "../lib/dateFormat";
import { useAuth } from "../context/AuthContext";
import type { Document } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusBadge } from "../components/invoice-form/StatusBadge";

const DOC_TYPE_SHORT: Record<string, string> = { "01": "FE", "91": "NC", "92": "ND", "05": "DS" };
const ROUTE_FOR_TYPE: Record<string, string> = {
  "01": "/documents/invoices", "91": "/documents/credit-notes",
  "92": "/documents/debit-notes", "05": "/documents/support-documents",
};

function money(cents: number): string {
  return formatCOP.format(cents / 100);
}

function trend(current: number, prev: number): { label: string; positive: boolean } | null {
  if (prev === 0 && current === 0) return null;
  if (prev === 0) return { label: "nuevo", positive: true };
  const pct = Math.round(((current - prev) / prev) * 100);
  return { label: `${pct > 0 ? "+" : ""}${pct}% vs. mes anterior`, positive: pct >= 0 };
}

export function ElectronicDashboardPage() {
  const { activeCompany } = useAuth();
  const navigate = useNavigate();

  const [documents, setDocuments] = useState<Document[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listDocuments({ from: addDaysColombiaISO(-395), limit: 200 })
      .then(setDocuments)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el panel de documentos"));
  }, []);

  const stats = useMemo(() => computeBillingStats(documents ?? []), [documents]);
  const cm = stats.current_month;
  const pm = stats.previous_month;
  const acceptRate = cm.document_count > 0 ? (cm.accepted_count / cm.document_count) * 100 : 0;
  const acceptRatePrev = pm.document_count > 0 ? (pm.accepted_count / pm.document_count) * 100 : 0;

  const recent = useMemo(
    () => [...(documents ?? [])].sort((a, b) => (b.issue_date ?? "").localeCompare(a.issue_date ?? "")).slice(0, 5),
    [documents],
  );

  const loading = documents === null;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Documentos" }]} />
      <div className="mb-3">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <FileText className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Panel de documentos electrónicos
        </h1>
        <p className="mt-0.5 text-xs text-(--text-secondary)">{activeCompany?.trade_name || activeCompany?.business_name}</p>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {loading ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : (
        <>
          <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Ingresos del mes <TrendingUp className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{money(cm.revenue_cents)}</div>
              {(() => { const t = trend(cm.revenue_cents, pm.revenue_cents); return t ? (
                <div className={`mt-1 text-xs ${t.positive ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>{t.label}</div>
              ) : <div className="mt-1 text-xs text-(--text-secondary)">Sin datos aún</div>; })()}
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Documentos emitidos <FileText className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{cm.document_count}</div>
              {(() => { const t = trend(cm.document_count, pm.document_count); return t ? (
                <div className={`mt-1 text-xs ${t.positive ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>{t.label}</div>
              ) : <div className="mt-1 text-xs text-(--text-secondary)">Sin datos aún</div>; })()}
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Tasa de aprobación <CheckCircle2 className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">
                {cm.document_count > 0 ? `${Math.round(acceptRate)}%` : "—"}
              </div>
              {(() => { const t = pm.document_count > 0 ? trend(acceptRate, acceptRatePrev) : null; return t ? (
                <div className={`mt-1 text-xs ${t.positive ? "text-(--color-success-text)" : "text-(--color-danger-text)"}`}>{t.label}</div>
              ) : <div className="mt-1 text-xs text-(--text-secondary)">aceptados / emitidos</div>; })()}
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Borradores pendientes <Clock className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{cm.draft_count}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">sin confirmar</div>
            </Card>
          </div>

          {stats.by_type.length > 0 && (
            <Card className="mb-4 p-4">
              <h2 className="mb-2 text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Por tipo este mes</h2>
              <div className="flex flex-wrap gap-4">
                {stats.by_type.map((t) => (
                  <button
                    key={t.type_code}
                    type="button"
                    onClick={() => navigate(ROUTE_FOR_TYPE[t.type_code] ?? "/documents/invoices")}
                    className="flex items-center gap-2 rounded border border-(--border-light) px-3 py-1.5 text-xs hover:bg-(--bg-hover)"
                  >
                    <span className="font-semibold text-(--text-primary)">{DOC_TYPE_SHORT[t.type_code] ?? t.type_code}</span>
                    <span className="text-(--text-secondary)">{t.count} · {money(t.revenue_cents)}</span>
                  </button>
                ))}
              </div>
            </Card>
          )}

          <Card className="p-4">
            <div className="mb-2 flex items-center justify-between">
              <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Actividad reciente</h2>
              <button type="button" onClick={() => navigate("/documents/invoices")} className="flex items-center gap-1 text-xs text-(--accent-primary) hover:underline">
                Ver todos <ArrowRight className="h-3 w-3" />
              </button>
            </div>
            {recent.length === 0 ? (
              <p className="py-4 text-center text-xs text-(--text-muted)">Aún no hay documentos emitidos</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="text-(--text-secondary)">
                    <tr>
                      <th className="py-1.5 pr-3 font-medium">Fecha</th>
                      <th className="py-1.5 pr-3 font-medium">Número</th>
                      <th className="py-1.5 pr-3 font-medium">Tipo</th>
                      <th className="py-1.5 pr-3 font-medium">Tercero</th>
                      <th className="py-1.5 pr-3 text-right font-medium">Total</th>
                      <th className="py-1.5 font-medium">Estado</th>
                    </tr>
                  </thead>
                  <tbody>
                    {recent.map((doc, i) => {
                      const cp = doc.dian_document_type_code === "05" ? doc.supplier : doc.customer;
                      return (
                        <tr
                          key={doc.id}
                          className={`cursor-pointer border-t border-(--border-light) hover:bg-(--bg-hover) ${i % 2 === 1 ? "bg-(--bg-secondary)" : ""}`}
                          onClick={() => navigate(`${ROUTE_FOR_TYPE[doc.dian_document_type_code] ?? "/documents/invoices"}/${doc.id}`)}
                        >
                          <td className="py-1.5 pr-3 text-(--text-secondary)">{formatDateOnly(doc.issue_date)}</td>
                          <td className="py-1.5 pr-3 font-mono text-(--text-primary)">{doc.prefix || "—"}{doc.number ?? ""}</td>
                          <td className="py-1.5 pr-3 text-(--text-primary)">{DOC_TYPE_SHORT[doc.dian_document_type_code] ?? doc.dian_document_type_code}</td>
                          <td className="max-w-45 truncate py-1.5 pr-3 text-(--text-secondary)">{cp?.name ?? "—"}</td>
                          <td className="py-1.5 pr-3 text-right font-mono text-(--text-primary)">{doc.totals ? money(doc.totals.payable_cents) : "—"}</td>
                          <td className="py-1.5"><StatusBadge status={doc.status} /></td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </>
      )}
    </div>
  );
}
