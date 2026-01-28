import { Spinner } from '@/shared/ui/Spinner';
import { MESSAGES } from '@/shared/messages';

export function LoadingSpinner({
  message = MESSAGES.common.loading.default,
}: {
  message?: string;
}) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-black text-emerald-50">
      <div className="flex flex-col items-center gap-3">
        <Spinner size="lg" borderColorClass="border-emerald-400" />
        <p className="text-xs text-emerald-500/80">{message}</p>
      </div>
    </div>
  );
}
