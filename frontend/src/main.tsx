import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./modules/app/App";
import { ToastProvider } from "./shared/ui/ToastProvider";
import { ErrorBoundary } from "./shared/ui/ErrorBoundary";
import "./styles/index.css";

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
        <App />
      </ToastProvider>
    </ErrorBoundary>
  </React.StrictMode>
);
