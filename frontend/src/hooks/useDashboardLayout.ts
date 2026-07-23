import { useCallback, useState } from "react";

export const WIDGET_IDS = [
  "w_revenue", "w_docs", "w_acceptance", "w_drafts",
  "w_chart_area", "w_chart_type", "w_ytd", "w_recent",
] as const;
export type WidgetId = (typeof WIDGET_IDS)[number];

export interface GridItem {
  i: WidgetId;
  x: number;
  y: number;
  w: number;
  h: number;
  minW?: number;
  minH?: number;
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

// Tamaño base de cada widget (12 columnas totales, rowHeight=40px)
const DEFAULT_SIZES: Record<WidgetId, { w: number; h: number }> = {
  w_revenue:    { w: 3, h: 4 },
  w_docs:       { w: 3, h: 4 },
  w_acceptance: { w: 3, h: 4 },
  w_drafts:     { w: 3, h: 4 },
  w_chart_area: { w: 9, h: 6 },
  w_chart_type: { w: 3, h: 6 },
  w_ytd:        { w: 12, h: 3 },
  w_recent:     { w: 12, h: 6 },
};

const DEFAULT_ORDER: WidgetId[] = [
  "w_revenue", "w_docs", "w_acceptance", "w_drafts",
  "w_chart_area", "w_chart_type",
  "w_ytd", "w_recent",
];

// Empaqueta widgets en filas de 12 columnas, izquierda-derecha
function packItems(
  orderedIds: WidgetId[],
  sizeOverrides: Partial<Record<WidgetId, { w: number; h: number }>>,
): GridItem[] {
  let x = 0, y = 0, rowH = 0;
  const result: GridItem[] = [];

  for (const id of orderedIds) {
    const { w, h } = { ...DEFAULT_SIZES[id], ...(sizeOverrides[id] ?? {}) };

    // Wrap si no cabe o si el widget es full-width
    if (x > 0 && (x + w > 12 || w >= 12)) {
      y += rowH;
      x = 0;
      rowH = 0;
    }

    result.push({ i: id, x, y, w, h, minW: 1, minH: 2 });
    x += w;
    rowH = Math.max(rowH, h);
  }

  return result;
}

// Deriva el orden visual (arriba→abajo, izquierda→derecha) desde posiciones
function positionsToOrder(items: GridItem[]): WidgetId[] {
  return [...items]
    .sort((a, b) => a.y !== b.y ? a.y - b.y : a.x - b.x)
    .map((it) => it.i);
}

// Extrae mapa de tamaños actuales de los items
function toSizeMap(items: GridItem[]): Partial<Record<WidgetId, { w: number; h: number }>> {
  return Object.fromEntries(items.map((it) => [it.i, { w: it.w, h: it.h }]));
}

interface DashboardLayout {
  items: GridItem[];   // todos los widgets (visibles + ocultos) con posiciones actuales
  hidden: WidgetId[];
}

const DEFAULT: DashboardLayout = {
  items: packItems(DEFAULT_ORDER, {}),
  hidden: [],
};

const LS_KEY = "dashboard_layout_v5";

function load(): DashboardLayout {
  try {
    const raw = localStorage.getItem(LS_KEY);
    if (!raw) return DEFAULT;
    const p = JSON.parse(raw) as Partial<DashboardLayout>;
    const known = new Set<string>(WIDGET_IDS);

    const storedItems = (p.items ?? []).filter(
      (it): it is GridItem => known.has(it.i),
    );
    const storedIds = new Set(storedItems.map((it) => it.i));
    const newWidgets = DEFAULT.items.filter((it) => !storedIds.has(it.i));

    return {
      items: [...storedItems, ...newWidgets],
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

  // Drag/resize de RGL → guardar posiciones directamente (sin repaquetear)
  const updateItems = useCallback(
    (newVisible: readonly { i: string; x: number; y: number; w: number; h: number }[]) => {
      const hiddenItems = layout.items.filter((it) => layout.hidden.includes(it.i));
      const merged: GridItem[] = [
        ...newVisible
          .filter((it): it is typeof it & { i: WidgetId } =>
            (WIDGET_IDS as readonly string[]).includes(it.i)
          )
          .map((it) => ({
            ...(layout.items.find((s) => s.i === it.i) ?? {}),
            i: it.i, x: it.x, y: it.y, w: it.w, h: it.h,
          })),
        ...hiddenItems,
      ];
      update({ ...layout, items: merged });
    },
    [layout, update],
  );

  // Ocultar → repaquetear visibles para cerrar el hueco
  const hide = useCallback(
    (id: WidgetId) => {
      const newHidden = [...layout.hidden, id];
      const visibleNow = layout.items.filter((it) => !newHidden.includes(it.i));
      const packed = packItems(positionsToOrder(visibleNow), toSizeMap(layout.items));
      const hiddenItems = layout.items.filter((it) => newHidden.includes(it.i));
      update({ items: [...packed, ...hiddenItems], hidden: newHidden });
    },
    [layout, update],
  );

  // Mostrar → añadir al final y repaquetear
  const show = useCallback(
    (id: WidgetId) => {
      const newHidden = layout.hidden.filter((h) => h !== id);
      const currentVisible = layout.items.filter((it) => !layout.hidden.includes(it.i));
      const order = [...positionsToOrder(currentVisible), id];
      const packed = packItems(order, toSizeMap(layout.items));
      const stillHidden = layout.items.filter((it) => newHidden.includes(it.i));
      update({ items: [...packed, ...stillHidden], hidden: newHidden });
    },
    [layout, update],
  );

  const reset = useCallback(() => update(DEFAULT), [update]);

  const visibleItems = layout.items.filter((it) => !layout.hidden.includes(it.i));

  return { visibleItems, hidden: layout.hidden, updateItems, hide, show, reset };
}
