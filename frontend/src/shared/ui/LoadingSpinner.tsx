import { Spinner } from '@/shared/ui/Spinner';
import { MESSAGES } from '@/shared/messages';

export function LoadingSpinner({
  message = MESSAGES.common.loading.default,
}: {
  message?: string;
}) {
  return (
    <div className="app-min-height flex items-center justify-center bg-[var(--color-canvas)] text-emerald-50">
      <div className="flex flex-col items-center gap-3" role="status">
        <Spinner size="lg" borderColorClass="border-emerald-400" />
        <p className="text-xs text-emerald-500/80">{message}</p>
      </div>
    </div>
  );
}
