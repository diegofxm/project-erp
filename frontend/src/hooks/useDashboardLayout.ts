import { useCallback, useState } from "react";

export const WIDGET_IDS = [
  "w_revenue", "w_docs", "w_acceptance", "w_drafts",
  "w_chart_area", "w_chart_type", "w_ytd", "w_recent",
] as const;
export type WidgetId = (typeof WIDGET_IDS)[number];

// Shape compatible con LayoutItem de react-grid-layout
export interface GridItem {
  i: WidgetId;
  x: number;
  y: number;
  w: number;
  h: number;
  minW?: number;
  minH?: number;
  maxW?: number;
  maxH?: number;
  isDraggable?: boolean;
  isResizable?: boolean;
  static?: boolean;
}

export const WIDGET_LABELS: Record<WidgetId, string> = {
  w_revenue:    "Ingresos del mes",
  w_docs:       "Documentos emitidos",
  w_acceptance: "Tasa de aprobación",
  w_drafts:     "Borradores pendientes",
  w_chart_area: "Tendencia de ingresos",
  w_chart_type: "Por tipo",
  w_ytd:        "Acumulado del año",
  w_recent:     "Actividad reciente",
};

// 12 columnas, rowHeight=40px
// w_revenue/docs/acceptance/drafts: 3 col cada uno → una fila de 4 KPIs
// w_chart_area: 9 col, w_chart_type: 3 col → segunda fila
// w_ytd + w_recent: 12 col → filas completas
const DEFAULT_ITEMS: GridItem[] = [
  { i: "w_revenue",    x: 0,  y: 0,  w: 3,  h: 4,  minW: 2, minH: 3 },
  { i: "w_docs",       x: 3,  y: 0,  w: 3,  h: 4,  minW: 2, minH: 3 },
  { i: "w_acceptance", x: 6,  y: 0,  w: 3,  h: 4,  minW: 2, minH: 3 },
  { i: "w_drafts",     x: 9,  y: 0,  w: 3,  h: 4,  minW: 2, minH: 3 },
  { i: "w_chart_area", x: 0,  y: 4,  w: 9,  h: 9,  minW: 3, minH: 5 },
  { i: "w_chart_type", x: 9,  y: 4,  w: 3,  h: 9,  minW: 2, minH: 5 },
  { i: "w_ytd",        x: 0,  y: 13, w: 12, h: 4,  minW: 3, minH: 3 },
  { i: "w_recent",     x: 0,  y: 17, w: 12, h: 8,  minW: 4, minH: 5 },
];

interface DashboardLayout {
  items: GridItem[];
  hidden: WidgetId[];
}

const DEFAULT: DashboardLayout = { items: DEFAULT_ITEMS, hidden: [] };
const LS_KEY = "dashboard_layout_v3";

function load(): DashboardLayout {
  try {
    const raw = localStorage.getItem(LS_KEY);
    if (!raw) return DEFAULT;
    const p = JSON.parse(raw) as Partial<DashboardLayout>;
    const known = new Set<string>(WIDGET_IDS);

    const storedItems = (p.items ?? []).filter(
      (it): it is GridItem => known.has(it.i),
    );
    // Append items added after the stored layout was saved
    for (const def of DEFAULT_ITEMS) {
      if (!storedItems.find((it) => it.i === def.i)) storedItems.push(def);
    }
    return {
      items: storedItems,
      hidden: (p.hidden ?? []).filter((id): id is WidgetId => known.has(id)),
    };
  } catch {
    return DEFAULT;
  }
}

function persist(l: DashboardLayout) {
  localStorage.setItem(LS_KEY, JSON.stringify(l));
}

export function useDashboardLayout() {
  const [layout, setLayout] = useState<DashboardLayout>(load);

  const update = useCallback((next: DashboardLayout) => {
    setLayout(next);
    persist(next);
  }, []);

  // Llamado por react-grid-layout en cada drag/resize — solo llega con items visibles.
  // Fusionamos con los items ocultos para no perder sus posiciones.
  const updateItems = useCallback(
    (newVisibleItems: readonly { i: string; x: number; y: number; w: number; h: number }[]) => {
      const hiddenItems = layout.items.filter((it) => layout.hidden.includes(it.i));
      const merged: GridItem[] = [
        ...newVisibleItems
          .filter((it): it is typeof it & { i: WidgetId } =>
            (WIDGET_IDS as readonly string[]).includes(it.i)
          )
          .map((it) => ({
            ...(layout.items.find((s) => s.i === it.i) ?? {}),
            i: it.i,
            x: it.x, y: it.y, w: it.w, h: it.h,
          })),
        ...hiddenItems,
      ];
      update({ ...layout, items: merged });
    },
    [layout, update],
  );

  const hide = useCallback(
    (id: WidgetId) => update({ ...layout, hidden: [...layout.hidden, id] }),
    [layout, update],
  );
  const show = useCallback(
    (id: WidgetId) => update({ ...layout, hidden: layout.hidden.filter((h) => h !== id) }),
    [layout, update],
  );
  const reset = useCallback(() => update(DEFAULT), [update]);

  const visibleItems = layout.items.filter((it) => !layout.hidden.includes(it.i));

  return { visibleItems, hidden: layout.hidden, updateItems, hide, show, reset };
}
