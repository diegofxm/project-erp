# Propuesta de diseño — v2, "Contabilidad primero"

Mockups estáticos (HTML + CSS + JS plano, sin build) para revisar una dirección visual nueva
antes de tocar `frontend/src`. No se ejecuta, no se compila, no depende del proyecto real —
ábrelos directamente en el navegador.

**v2 (esta versión)**: el navbar y el sidebar son un calco 1:1 de los componentes reales
(`Navbar.tsx`, `Sidebar.tsx`, `SubNav.tsx` — mismas alturas, mismo ancho, mismos nombres de
variable CSS, mismos ítems de navegación). Lo único que cambia ahí es el color: azul oscuro en
vez del slate/azul genérico de hoy. El sub-nav de pestañas (Panel / Plan de cuentas / Asientos /
Períodos / Informes) usa el mismo patrón que ya existe para Documentos/Configuración/Admin — no
es un componente nuevo. Todo lo demás (kanban, notebook, stat-buttons, listas) es contenido
nuevo para pantallas que hoy no existen.

## Cómo verla

Abre `index.html` (doble clic, o arrástralo a una pestaña del navegador). Desde ahí hay enlaces
a los demás mockups. El botón de hamburguesa del navbar colapsa/expande el sidebar, igual que
en la app real.

## Archivos

| Archivo | Qué muestra |
|---|---|
| `index.html` | Portada de la propuesta: por qué cambiar, comparación antes/después, qué trae, enlaces |
| `dashboard.html` | Panel de Contabilidad — kanban de diarios (Ventas/Compras/Banco/Miscelánea) + KPIs |
| `chart-of-accounts.html` | Plan único de cuentas — lista agrupable por clase (PUC), con filtros |
| `journal-entry.html` | Un asiento contable — notebook de pestañas, botones-stat, barra de estado |
| `components.html` | Guía de estilo — paleta, tipografía, botones, badges, pestañas, etc. |
| `shared.css` | Tokens (mismos nombres que `frontend/src/index.css`) y clases de componentes nuevos |
| `shared.js` | Colapso de sidebar, pestañas del notebook, grupos colapsables de la lista — vanilla JS |

## Decisiones de diseño

- **Navbar y sidebar: sin cambios de estructura**, solo de color. `--navbar-bg` y
  `--accent-primary` pasan a azul oscuro (`#14294a` / `#2f6fb0`); todo lo demás
  (`--bg-primary`, `--text-primary`, `--border-color`, radios, tamaños) queda igual que hoy.
- **Tipografía sin cambios**: misma stack de sistema, misma base de 13px. No se introduce
  ninguna fuente nueva.
- **Semántica de estado fija**: éxito/alerta/peligro/info son exactamente los mismos colores
  que ya existen en `index.css` (`--color-success`, `--color-danger`, etc.), no dependen del
  acento.
- **Patrones de Odoo, solo en el contenido nuevo**: breadcrumb, botones-stat, barra de estado
  (stepper), notebook de pestañas, tarjetas kanban, listas con barra de herramientas y grupos
  colapsables — todo esto vive dentro del área de contenido (`.main`), nunca en el shell.
- **Nuevo en el sidebar**: un ítem "Contabilidad" (no existía porque no había UI de contabilidad
  hasta ahora), con su propio sub-nav de 5 pestañas siguiendo el mismo patrón que ya usan
  Documentos/Configuración/Admin.

## Si se aprueba

1. Cambiar los valores de `--navbar-bg` y `--accent-primary` (y derivar `--bg-selected`) en
   `frontend/src/index.css` — es literalmente ese archivo, mismo mecanismo de variables CSS.
2. Agregar el ítem "Contabilidad" a `NAV_ITEMS` en `Sidebar.tsx` y su entrada en `SUB_NAVS` en
   `SubNav.tsx` (mismo patrón que ya usan `/documents` y `/settings`).
3. Construir los componentes nuevos en React como piezas compartidas: `<Notebook>`,
   `<StatButton>`, `<StatusStepper>`, `<KanbanCard>`, `<ListToolbar>`.
4. Primera pantalla real candidata: dashboard de Contabilidad — el módulo
   `erp/internal/accounting` ya tiene API completa (PUC, períodos, diario, balance, P&G) pero
   cero UI hoy.

Ver también la memoria de proyecto `project_ubl_status.md` para el estado real del backend de
cada módulo mencionado aquí.
