export interface TabItem {
  id: string;
  label: string;
  disabled?: boolean;
}

interface TabsProps {
  tabs: TabItem[];
  activeId: string;
  onChange: (id: string) => void;
}

// Barra de pestañas tipo IDE (sección 7 del design system): h-9, pestaña activa con borde
// inferior de 2px en el acento, inactiva sin borde. Pestañas deshabilitadas usan el mismo
// tratamiento "próximamente" que Sidebar.tsx ya aplica a sus ítems futuros.
export function Tabs({ tabs, activeId, onChange }: TabsProps) {
  return (
    <div className="flex h-9 items-center border-b border-(--border-light)">
      {tabs.map((tab) => {
        const isActive = tab.id === activeId;
        return (
          <button
            key={tab.id}
            type="button"
            disabled={tab.disabled}
            title={tab.disabled ? `${tab.label} (próximamente)` : undefined}
            onClick={() => onChange(tab.id)}
            className={`h-full border-b-2 px-3 text-xs font-medium transition-colors ${
              tab.disabled
                ? "cursor-not-allowed border-transparent text-(--text-muted) opacity-60"
                : isActive
                  ? "border-(--accent-primary) text-(--text-primary)"
                  : "border-transparent text-(--text-secondary) hover:text-(--text-primary)"
            }`}
          >
            {tab.label}
          </button>
        );
      })}
    </div>
  );
}
