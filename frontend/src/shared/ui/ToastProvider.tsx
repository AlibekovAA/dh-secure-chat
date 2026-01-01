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
      <div className="pointer-events-none fixed right-4 top-4 z-50 flex flex-col gap-2 max-w-sm">
        {toasts.map((toast, index) => (
          <div
            key={toast.id}
            className="pointer-events-auto animate-toast-enter"
            style={{
              animationDelay: `${index * 50}ms`,
            }}
          >
            <div
              className={`w-full rounded-lg border px-4 py-3 text-sm shadow-xl backdrop-blur-md smooth-transition ${
                toast.kind === "error"
                  ? "border-red-500/50 bg-red-900/90 text-red-50"
                  : toast.kind === "success"
                    ? "border-emerald-500/50 bg-emerald-900/90 text-emerald-50"
                    : "border-yellow-400/50 bg-yellow-900/90 text-yellow-50"
              }`}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-start gap-2 flex-1 min-w-0">
                  {toast.kind !== "error" && (
                    <div className={`flex-shrink-0 mt-0.5 ${
                      toast.kind === "success"
                        ? "text-emerald-400"
                        : "text-yellow-400"
                    }`}>
                      {toast.kind === "success" ? (
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                        </svg>
                      ) : (
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.785-1.25 2.785-2.8V8.8c0-1.55-1.245-2.8-2.785-2.8H5.062C3.522 6 2.277 7.25 2.277 8.8v8.4c0 1.55 1.245 2.8 2.785 2.8z" />
                        </svg>
                      )}
                    </div>
                  )}
                  <span className="flex-1 break-words leading-relaxed">{toast.message}</span>
                </div>
                <button
                  onClick={() => removeToast(toast.id)}
                  className="flex-shrink-0 -mt-1 -mr-1 h-6 w-6 rounded hover:bg-black/30 flex items-center justify-center transition-colors opacity-70 hover:opacity-100"
                  aria-label="Закрыть"
                >
                  <svg
                    className="w-4 h-4"
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
