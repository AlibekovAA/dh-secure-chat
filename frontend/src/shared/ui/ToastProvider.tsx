import { createContext, ReactNode, useContext, useMemo, useState } from "react";

type ToastKind = "success" | "error";

type Toast = {
  id: number;
  message: string;
  kind: ToastKind;
};

type ToastContextValue = {
  showToast(message: string, kind?: ToastKind): void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

type Props = {
  children: ReactNode;
};

export function ToastProvider({ children }: Props) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const value = useMemo<ToastContextValue>(
    () => ({
      showToast(message, kind = "error") {
        setToasts(current => {
          const id = (current.at(-1)?.id ?? 0) + 1;
          return [...current, { id, message, kind }];
        });

        setTimeout(() => {
          setToasts(current => current.slice(1));
        }, 3500);
      }
    }),
    []
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="pointer-events-none fixed right-4 top-4 z-50 flex flex-col gap-2">
        {toasts.map(toast => (
          <div
            key={toast.id}
            className={`pointer-events-auto w-72 rounded-lg border px-4 py-3 text-sm shadow-lg backdrop-blur ${
              toast.kind === "error"
                ? "border-red-500/40 bg-red-900/80 text-red-50"
                : "border-emerald-500/40 bg-emerald-900/80 text-emerald-50"
            }`}
          >
            {toast.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return ctx;
}
