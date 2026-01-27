import { Component, type ReactNode } from 'react';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: (error: Error, reset: () => void) => ReactNode;
  onError?: (error: Error, errorInfo: { componentStack: string }) => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: { componentStack: string }): void {
    this.props.onError?.(error, errorInfo);

    if (
      error.message?.includes('Failed to fetch dynamically imported module') ||
      error.message?.includes('Failed to load resource') ||
      error.message?.includes('404')
    ) {
      setTimeout(() => {
        const shouldReload = window.confirm(
          'Обнаружена устаревшая версия приложения. Перезагрузить страницу?'
        );
        if (shouldReload) {
          window.location.reload();
        }
      }, 100);
    }
  }

  reset = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render(): ReactNode {
    if (this.state.hasError && this.state.error) {
      if (this.props.fallback) {
        return this.props.fallback(this.state.error, this.reset);
      }
      const isModuleLoadError =
        this.state.error.message?.includes(
          'Failed to fetch dynamically imported module'
        ) ||
        this.state.error.message?.includes('Failed to load resource') ||
        this.state.error.message?.includes('404');

      return (
        <div className="min-h-screen flex items-center justify-center bg-black text-emerald-50">
          <div className="flex flex-col items-center gap-4 p-6 max-w-md">
            <div className="text-red-400 text-xl font-semibold">
              {isModuleLoadError
                ? 'Устаревшая версия приложения'
                : 'Произошла ошибка'}
            </div>
            <div className="text-sm text-gray-400 text-center">
              {isModuleLoadError
                ? 'Обнаружена устаревшая версия приложения. Пожалуйста, перезагрузите страницу.'
                : this.state.error.message || 'Неизвестная ошибка'}
            </div>
            <div className="flex gap-2">
              {isModuleLoadError ? (
                <button
                  onClick={() => window.location.reload()}
                  className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded transition-colors"
                >
                  Перезагрузить страницу
                </button>
              ) : (
                <button
                  onClick={this.reset}
                  className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded transition-colors"
                >
                  Попробовать снова
                </button>
              )}
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
