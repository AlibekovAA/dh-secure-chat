type Props = {
  filename: string;
  mimeType: string;
  size: number;
  blob?: Blob;
  isOwn: boolean;
  accessMode?: 'download_only' | 'view_only' | 'both';
  onDownloadStateChange?: (active: boolean) => void;
  onView?: () => void;
};

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function getFileIcon(mimeType: string): string {
  if (mimeType.startsWith('image/')) return '🖼️';
  if (mimeType === 'application/pdf') return '📄';
  if (mimeType.includes('word') || mimeType.includes('document')) return '📝';
  if (mimeType.includes('sheet') || mimeType.includes('excel')) return '📊';
  if (mimeType.includes('presentation') || mimeType.includes('powerpoint')) return '📽️';
  if (mimeType === 'text/plain') return '📃';
  return '📎';
}

export function FileMessage({
  filename,
  mimeType,
  size,
  blob,
  isOwn,
  accessMode = 'both',
  onDownloadStateChange,
  onView,
}: Props) {
  const canDownload = isOwn || accessMode === 'download_only' || accessMode === 'both';
  const canView = isOwn || accessMode === 'view_only' || accessMode === 'both';

  const handleDownload = () => {
    if (!blob) return;

    onDownloadStateChange?.(true);
    try {
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } finally {
      onDownloadStateChange?.(false);
    }
  };

  return (
    <>
      <div className="space-y-2">
        <div
          className={`flex items-center gap-3 p-3 rounded-lg border ${
            isOwn
              ? 'bg-emerald-500/20 border-emerald-500/40'
              : 'bg-emerald-900/20 border-emerald-700/40'
          }`}
        >
          <span className="text-2xl">{getFileIcon(mimeType)}</span>
          <div className="flex-1 min-w-0">
            <p
              className={`text-sm font-medium truncate ${
                isOwn ? 'text-emerald-50' : 'text-emerald-100'
              }`}
            >
              {filename}
            </p>
            <p className="text-xs text-emerald-500/80 mt-0.5">
              {formatFileSize(size)}
            </p>
          </div>
        </div>

        {blob && (
          <div className="flex items-center justify-end gap-2">
            {canView && (
              <button
                type="button"
                onClick={onView}
                className="text-xs text-emerald-400 hover:text-emerald-200 transition-colors flex items-center gap-1 px-2 py-1 rounded bg-emerald-900/20 hover:bg-emerald-900/40 border border-emerald-700/40"
                title="Просмотреть файл"
              >
                <svg
                  className="w-3.5 h-3.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                  />
                </svg>
                Просмотреть
              </button>
            )}
            {canDownload && (
              <button
                type="button"
                onClick={handleDownload}
                className="text-xs text-emerald-400 hover:text-emerald-200 transition-colors flex items-center gap-1 px-2 py-1 rounded bg-emerald-900/20 hover:bg-emerald-900/40 border border-emerald-700/40"
                title="Скачать файл"
              >
                <svg
                  className="w-3.5 h-3.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                  />
                </svg>
                Скачать
              </button>
            )}
          </div>
        )}
      </div>
    </>
  );
}
