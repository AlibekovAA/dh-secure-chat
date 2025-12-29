import React, { lazy, Suspense } from "react";
import ReactDOM from "react-dom/client";
import { ToastProvider } from "./shared/ui/ToastProvider";
import { ErrorBoundary } from "./shared/ui/ErrorBoundary";
import "./styles/index.css";

const App = lazy(() => import("./modules/app/App").then((module) => ({ default: module.App })));

window.addEventListener('error', (event) => {
  if (event.target && (event.target as HTMLElement).tagName === 'IMG') {
    const img = event.target as HTMLImageElement;
    if (img.src && img.src.startsWith('blob:')) {
      event.preventDefault();
      return false;
    }
  }
  if (event.target && (event.target as HTMLElement).tagName === 'AUDIO') {
    const audio = event.target as HTMLAudioElement;
    if (audio.src && audio.src.startsWith('blob:')) {
      const error = audio.error;
      if (error && (error.code === 4 || error.code === 2)) {
        event.preventDefault();
        return false;
      }
    }
  }
  return true;
}, true);

window.addEventListener('unhandledrejection', (event) => {
  if (event.reason && typeof event.reason === 'string' && event.reason.includes('blob:')) {
    event.preventDefault();
  }
});

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <ErrorBoundary>
      <ToastProvider>
        <Suspense
          fallback={
            <div className="min-h-screen flex items-center justify-center bg-black text-emerald-50">
              <div className="flex flex-col items-center gap-3">
                <div className="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
                <p className="text-xs text-emerald-500/80">Загрузка...</p>
              </div>
            </div>
          }
        >
          <App />
        </Suspense>
      </ToastProvider>
    </ErrorBoundary>
  </React.StrictMode>
);
