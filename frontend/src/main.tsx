import React, { lazy, Suspense } from 'react';
import ReactDOM from 'react-dom/client';
import { ToastProvider } from '@/shared/ui/ToastProvider';
import { ErrorBoundary } from '@/shared/ui/ErrorBoundary';
import { LoadingSpinner } from '@/shared/ui/LoadingSpinner';
import { retryImport } from '@/shared/utils/retry-import';
import '@/styles/index.css';

const App = lazy(() =>
  retryImport(() =>
    import('@/modules/app/App').then((module) => ({ default: module.App }))
  )
);

window.addEventListener(
  'error',
  (event) => {
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
  },
  true
);

window.addEventListener('unhandledrejection', (event) => {
  if (
    event.reason &&
    typeof event.reason === 'string' &&
    event.reason.includes('blob:')
  ) {
    event.preventDefault();
    return;
  }

  if (
    event.reason &&
    (event.reason instanceof TypeError || event.reason instanceof Error) &&
    (event.reason.message?.includes(
      'Failed to fetch dynamically imported module'
    ) ||
      event.reason.message?.includes('Failed to load resource') ||
      event.reason.message?.includes('404'))
  ) {
    event.preventDefault();
    const shouldReload = window.confirm(
      'Обнаружена устаревшая версия приложения. Перезагрузить страницу?'
    );
    if (shouldReload) {
      window.location.reload();
    }
  }
});

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <ErrorBoundary>
      <ToastProvider>
        <Suspense fallback={<LoadingSpinner />}>
          <App />
        </Suspense>
      </ToastProvider>
    </ErrorBoundary>
  </React.StrictMode>
);
