import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from "react";
import { Toast, type ToastItem, type ToastTone } from "../components/ui/Toast";

// Éxito se lee rápido y desaparece pronto; error se queda más tiempo porque suele traer más
// texto y el usuario necesita poder leerlo con calma antes de que se vaya solo.
const DURATION_MS: Record<ToastTone, number> = { success: 4000, danger: 6000 };

interface ToastContextValue {
  success: (message: string) => void;
  error: (message: string) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);
  const nextId = useRef(0);

  const dismiss = useCallback((id: number) => {
    setItems((current) => current.filter((t) => t.id !== id));
  }, []);

  const show = useCallback(
    (tone: ToastTone, message: string) => {
      const id = nextId.current++;
      setItems((current) => [...current, { id, tone, message }]);
      setTimeout(() => dismiss(id), DURATION_MS[tone]);
    },
    [dismiss],
  );

  const value = useMemo<ToastContextValue>(
    () => ({
      success: (message: string) => show("success", message),
      error: (message: string) => show("danger", message),
    }),
    [show],
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="fixed top-3 right-3 z-50 flex flex-col gap-2">
        {items.map((item) => (
          <Toast key={item.id} tone={item.tone} message={item.message} onClose={() => dismiss(item.id)} />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast debe usarse dentro de <ToastProvider>");
  return ctx;
}
