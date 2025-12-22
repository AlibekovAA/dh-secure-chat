import { useState, useMemo, useEffect } from 'react';

type Props = {
  filename: string;
  mimeType: string;
  size: number;
  blob?: Blob;
  isOwn: boolean;
  onDownloadStateChange?: (active: boolean) => void;
};

const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'image/gif'];
const MAX_PREVIEW_SIZE = 5 * 1024 * 1024;

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
  onDownloadStateChange,
}: Props) {
  const [imageError, setImageError] = useState(false);
  const [imageUrl, setImageUrl] = useState<string | null>(null);

  const isImage = useMemo(() => IMAGE_TYPES.includes(mimeType), [mimeType]);
  const canPreview = useMemo(
    () => isImage && blob && size <= MAX_PREVIEW_SIZE,
    [isImage, blob, size],
  );

  useEffect(() => {
    if (canPreview && blob && !imageUrl) {
      const url = URL.createObjectURL(blob);
      setImageUrl(url);
      return () => URL.revokeObjectURL(url);
    }
  }, [canPreview, blob, imageUrl]);

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
    <div className="space-y-2">
      {canPreview && imageUrl && !imageError ? (
        <div className="rounded-lg overflow-hidden border border-emerald-700/40 max-w-full bg-black/40">
          <img
            src={imageUrl}
            alt={filename}
            className="max-w-full max-h-64 object-contain"
            onError={(e) => {
              const img = e.currentTarget;
              if (img.src && img.src.startsWith('blob:')) {
                setImageError(true);
              }
            }}
          />
        </div>
      ) : (
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
      )}

      {!canPreview && (
        <div className="flex items-center justify-between">
          <p className="text-xs text-emerald-500/60 truncate flex-1 mr-2">
            {filename}
          </p>
          {blob && (
            <button
              type="button"
              onClick={handleDownload}
              className="text-xs text-emerald-400 hover:text-emerald-200 transition-colors flex items-center gap-1 flex-shrink-0"
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

      {canPreview && blob && (
        <div className="flex items-center justify-end mt-2">
          <button
            type="button"
            onClick={handleDownload}
            className="text-xs text-emerald-400 hover:text-emerald-200 transition-colors flex items-center gap-1"
            title="Скачать изображение"
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
        </div>
      )}
    </div>
  );
}
