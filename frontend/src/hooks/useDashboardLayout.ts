import { useCallback, useState } from "react";

export const SECTION_IDS = ["kpis", "charts", "ytd", "recent_docs"] as const;
export type SectionId = (typeof SECTION_IDS)[number];

export const KPI_IDS = ["kpi_revenue", "kpi_docs", "kpi_acceptance", "kpi_drafts"] as const;
export type KpiId = (typeof KPI_IDS)[number];

export const SECTION_LABELS: Record<SectionId, string> = {
  kpis: "Indicadores",
  charts: "Gráficas",
  ytd: "Acumulado del año",
  recent_docs: "Actividad reciente",
};

export const KPI_LABELS: Record<KpiId, string> = {
  kpi_revenue: "Ingresos del mes",
  kpi_docs: "Documentos emitidos",
  kpi_acceptance: "Tasa de aprobación",
  kpi_drafts: "Borradores pendientes",
};

interface Layout {
  sectionOrder: SectionId[];
  hiddenSections: SectionId[];
  hiddenKpis: KpiId[];
}

const DEFAULT: Layout = {
  sectionOrder: ["kpis", "charts", "ytd", "recent_docs"],
  hiddenSections: [],
  hiddenKpis: [],
};

const LS_KEY = "dashboard_layout_v1";

function load(): Layout {
  try {
    const raw = localStorage.getItem(LS_KEY);
    if (!raw) return DEFAULT;
    const p = JSON.parse(raw) as Partial<Layout>;
    const secSet = new Set<string>(SECTION_IDS);
    const kpiSet = new Set<string>(KPI_IDS);

    const sectionOrder = (p.sectionOrder ?? [...DEFAULT.sectionOrder]).filter(
      (s): s is SectionId => secSet.has(s),
    );
    // Append newly added sections not yet in stored order
    for (const s of SECTION_IDS) {
      if (!sectionOrder.includes(s)) sectionOrder.push(s);
    }
    return {
      sectionOrder,
      hiddenSections: (p.hiddenSections ?? []).filter((s): s is SectionId => secSet.has(s)),
      hiddenKpis: (p.hiddenKpis ?? []).filter((k): k is KpiId => kpiSet.has(k)),
    };
  } catch {
    return DEFAULT;
  }
}

function persist(l: Layout) {
  localStorage.setItem(LS_KEY, JSON.stringify(l));
}

export function useDashboardLayout() {
  const [layout, setLayout] = useState<Layout>(load);

  const update = useCallback((next: Layout) => {
    setLayout(next);
    persist(next);
  }, []);

  const reorderSections = useCallback(
    (ids: SectionId[]) => update({ ...layout, sectionOrder: ids }),
    [layout, update],
  );
  const hideSection = useCallback(
    (id: SectionId) => update({ ...layout, hiddenSections: [...layout.hiddenSections, id] }),
    [layout, update],
  );
  const showSection = useCallback(
    (id: SectionId) =>
      update({ ...layout, hiddenSections: layout.hiddenSections.filter((s) => s !== id) }),
    [layout, update],
  );
  const hideKpi = useCallback(
    (id: KpiId) => update({ ...layout, hiddenKpis: [...layout.hiddenKpis, id] }),
    [layout, update],
  );
  const showKpi = useCallback(
    (id: KpiId) =>
      update({ ...layout, hiddenKpis: layout.hiddenKpis.filter((k) => k !== id) }),
    [layout, update],
  );
  const reset = useCallback(() => update(DEFAULT), [update]);

  return {
    sectionOrder: layout.sectionOrder,
    visibleSections: layout.sectionOrder.filter((s) => !layout.hiddenSections.includes(s)),
    visibleKpis: KPI_IDS.filter((k) => !layout.hiddenKpis.includes(k)),
    hiddenSections: layout.hiddenSections,
    hiddenKpis: layout.hiddenKpis,
    reorderSections,
    hideSection,
    showSection,
    hideKpi,
    showKpi,
    reset,
  };
}
