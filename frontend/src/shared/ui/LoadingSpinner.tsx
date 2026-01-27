import { Spinner } from '@/shared/ui/Spinner';

export function LoadingSpinner({
  message = 'Загрузка...',
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
