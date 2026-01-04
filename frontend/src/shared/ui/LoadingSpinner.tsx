export function LoadingSpinner({ message = "Загрузка..." }: { message?: string }) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-black text-emerald-50">
      <div className="flex flex-col items-center gap-3">
        <div className="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
        <p className="text-xs text-emerald-500/80">{message}</p>
      </div>
    </div>
  );
}
