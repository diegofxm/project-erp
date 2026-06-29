import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { ConfirmDialog, type ConfirmRequest } from "../components/ui/ConfirmDialog";

interface PendingConfirm extends ConfirmRequest {
  resolve: (confirmed: boolean) => void;
}

interface ConfirmContextValue {
  // Mismo contrato que window.confirm (se resuelve a true/false) — migrar un sitio es
  // mecánico: if (!window.confirm("...")) return; -> if (!(await confirm("..."))) return;
  confirm: (message: string, options?: Omit<ConfirmRequest, "message">) => Promise<boolean>;
}

const ConfirmContext = createContext<ConfirmContextValue | null>(null);

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [pending, setPending] = useState<PendingConfirm | null>(null);

  const confirm = useCallback((message: string, options?: Omit<ConfirmRequest, "message">) => {
    return new Promise<boolean>((resolve) => {
      setPending({ message, ...options, resolve });
    });
  }, []);

  const value = useMemo<ConfirmContextValue>(() => ({ confirm }), [confirm]);

  function settle(confirmed: boolean) {
    pending?.resolve(confirmed);
    setPending(null);
  }

  return (
    <ConfirmContext.Provider value={value}>
      {children}
      {pending && (
        <ConfirmDialog
          message={pending.message}
          title={pending.title}
          tone={pending.tone}
          confirmLabel={pending.confirmLabel}
          onConfirm={() => settle(true)}
          onCancel={() => settle(false)}
        />
      )}
    </ConfirmContext.Provider>
  );
}

export function useConfirm(): ConfirmContextValue["confirm"] {
  const ctx = useContext(ConfirmContext);
  if (!ctx) throw new Error("useConfirm debe usarse dentro de <ConfirmProvider>");
  return ctx.confirm;
}
