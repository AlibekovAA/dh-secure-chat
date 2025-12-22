import { createContext, ReactNode, useContext, useMemo, useState } from "react";

type ToastKind = "success" | "error" | "warning";

type Toast = {
  id: number;
  message: string;
  kind: ToastKind;
};

type ToastContextValue = {
  showToast(message: string, kind?: ToastKind): void;
  removeToast(id: number): void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

type Props = {
  children: ReactNode;
};

export function ToastProvider({ children }: Props) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const removeToast = (id: number) => {
    setToasts(current => current.filter(toast => toast.id !== id));
  };

  const value = useMemo<ToastContextValue>(
    () => ({
      showToast(message, kind = "error") {
        setToasts(current => {
          const id = (current.at(-1)?.id ?? 0) + 1;
          const newToast = { id, message, kind };

          setTimeout(() => {
            setToasts(prev => prev.filter(toast => toast.id !== id));
          }, 3500);

          return [...current, newToast];
        });
      },
      removeToast
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
            className={`pointer-events-auto w-72 rounded-lg border px-4 py-3 text-sm shadow-lg backdrop-blur-sm toast-enter smooth-transition ${
              toast.kind === "error"
                ? "border-red-500/40 bg-red-900/80 text-red-50 glow-emerald-hover"
                : toast.kind === "success"
                  ? "border-emerald-500/40 bg-emerald-900/80 text-emerald-50 glow-emerald-hover"
                  : "border-yellow-400/40 bg-yellow-900/80 text-yellow-50 glow-emerald-hover"
            }`}
          >
            <div className="flex items-start justify-between gap-2">
              <span className="flex-1">{toast.message}</span>
              <button
                onClick={() => removeToast(toast.id)}
                className="flex-shrink-0 -mt-1 -mr-1 h-5 w-5 rounded hover:bg-black/20 flex items-center justify-center transition-colors"
                aria-label="Закрыть"
              >
                <svg
                  className="w-3 h-3"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>
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
