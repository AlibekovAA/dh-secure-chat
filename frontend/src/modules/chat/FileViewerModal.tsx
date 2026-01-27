import { useEffect, useState, useRef } from 'react';
import {
  VIDEO_MIME_TYPES,
  IMAGE_MIME_TYPES,
  PDF_MIME_TYPE,
  TEXT_MIME_TYPES,
} from '@/shared/constants';
import { Spinner } from '@/shared/ui/Spinner';

type Props = {
  filename: string;
  mimeType: string;
  blob: Blob;
  onClose: () => void;
  protected?: boolean;
};

export function FileViewerModal({
  filename,
  mimeType,
  blob,
  onClose,
  protected: isProtected = false,
}: Props) {
  const [objectUrl, setObjectUrl] = useState<string | null>(null);
  const [textContent, setTextContent] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [videoLoading, setVideoLoading] = useState(true);
  const [videoError, setVideoError] = useState(false);
  const [videoProgress, setVideoProgress] = useState(0);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const imageRef = useRef<HTMLImageElement | null>(null);
  const videoRef = useRef<HTMLVideoElement | null>(null);

  useEffect(() => {
    const isVideo = VIDEO_MIME_TYPES.some((type) => mimeType.includes(type));

    if (isVideo && blob) {
      const url = URL.createObjectURL(blob);
      setObjectUrl(url);
      return () => {
        URL.revokeObjectURL(url);
      };
    }

    if (IMAGE_MIME_TYPES.includes(mimeType)) {
      if (isProtected && canvasRef.current) {
        const img = new Image();
        const url = URL.createObjectURL(blob);
        img.onload = () => {
          const canvas = canvasRef.current;
          if (!canvas) return;

          const ctx = canvas.getContext('2d');
          if (!ctx) return;

          const containerWidth = window.innerWidth * 0.95 - 48;
          const containerHeight = window.innerHeight * 0.95 - 120;
          let width = img.width;
          let height = img.height;

          const scaleX = containerWidth / width;
          const scaleY = containerHeight / height;
          const scale = Math.min(scaleX, scaleY, 1);

          width = width * scale;
          height = height * scale;

          canvas.width = width;
          canvas.height = height;

          ctx.drawImage(img, 0, 0, width, height);

          if (isProtected) {
            ctx.fillStyle = 'rgba(0, 0, 0, 0.3)';
            ctx.fillRect(0, 0, width, height);

            ctx.save();
            ctx.globalAlpha = 0.5;
            ctx.fillStyle = 'rgba(16, 185, 129, 0.6)';
            ctx.font = `${Math.max(16, width / 20)}px Arial`;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';

            const watermark = 'Secure Chat - View Only';
            const x = width / 2;
            const y = height / 2;

            ctx.fillText(watermark, x, y);
            ctx.restore();
          }

          imageRef.current = img;
          URL.revokeObjectURL(url);
        };
        img.onerror = () => {
          setError('Не удалось загрузить изображение');
          URL.revokeObjectURL(url);
        };
        img.src = url;
      } else {
        const url = URL.createObjectURL(blob);
        setObjectUrl(url);
        return () => URL.revokeObjectURL(url);
      }
    } else if (mimeType === PDF_MIME_TYPE) {
      const url = URL.createObjectURL(blob);
      setObjectUrl(url);
      return () => URL.revokeObjectURL(url);
    } else if (TEXT_MIME_TYPES.includes(mimeType)) {
      blob
        .text()
        .then((text: string) => setTextContent(text))
        .catch((_err: unknown) => {
          setError('Не удалось прочитать файл');
        });
    } else {
      setError('Просмотр этого типа файла не поддерживается');
    }
  }, [blob, mimeType, isProtected]);

  useEffect(() => {
    if (!isProtected) return;

    const handleContextMenu = (e: MouseEvent) => {
      e.preventDefault();
      return false;
    };

    const handleKeyDown = (e: KeyboardEvent) => {
      if (
        e.key === 'F12' ||
        (e.ctrlKey && e.shiftKey && e.key === 'I') ||
        (e.ctrlKey && e.shiftKey && e.key === 'J') ||
        (e.ctrlKey && e.shiftKey && e.key === 'C') ||
        (e.ctrlKey && e.key === 'S') ||
        (e.ctrlKey && e.key === 'P') ||
        e.key === 'PrintScreen' ||
        (e.altKey && e.key === 'PrintScreen')
      ) {
        e.preventDefault();
        return false;
      }
    };

    const handleSelectStart = (e: Event) => {
      e.preventDefault();
      return false;
    };

    const handleDragStart = (e: DragEvent) => {
      e.preventDefault();
      return false;
    };

    const handleVisibilityChange = () => {
      if (document.hidden) {
        onClose();
      }
    };

    const handleBlur = () => {
      if (isProtected) {
        setTimeout(() => {
          if (!document.hasFocus()) {
            onClose();
          }
        }, 100);
      }
    };

    document.addEventListener('contextmenu', handleContextMenu);
    document.addEventListener('keydown', handleKeyDown);
    document.addEventListener('selectstart', handleSelectStart);
    document.addEventListener('dragstart', handleDragStart);
    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('blur', handleBlur);

    return () => {
      document.removeEventListener('contextmenu', handleContextMenu);
      document.removeEventListener('keydown', handleKeyDown);
      document.removeEventListener('selectstart', handleSelectStart);
      document.removeEventListener('dragstart', handleDragStart);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('blur', handleBlur);
    };
  }, [isProtected, onClose]);

  const handleDownload = () => {
    if (!objectUrl && !blob) return;
    const url = objectUrl || URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    if (!objectUrl) {
      URL.revokeObjectURL(url);
    }
  };

  const renderContent = () => {
    if (error) {
      return (
        <div className="flex items-center justify-center h-full">
          <div className="text-center">
            <div className="bg-red-900/20 border border-red-700/40 rounded-lg px-4 py-3 mb-4 max-w-md mx-auto">
              <div className="flex items-center gap-2 justify-center mb-2">
                <svg
                  className="w-5 h-5 text-red-400"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                <p className="text-sm font-medium text-red-300">{error}</p>
              </div>
            </div>
            <button
              onClick={handleDownload}
              className="px-4 py-2 text-sm font-medium bg-gradient-to-r from-emerald-500 to-emerald-400 hover:from-emerald-400 hover:to-emerald-300 text-black rounded-md transition-all duration-200 hover:scale-105 active:scale-95 shadow-lg shadow-emerald-500/30"
              style={{ willChange: 'transform' }}
            >
              Скачать файл
            </button>
          </div>
        </div>
      );
    }

    if (IMAGE_MIME_TYPES.includes(mimeType)) {
      if (isProtected && canvasRef.current) {
        return (
          <div
            className="flex items-center justify-center h-full w-full bg-black/40 overflow-auto p-4"
            style={{ userSelect: 'none' }}
          >
            <canvas
              ref={canvasRef}
              className="max-w-full max-h-full"
              style={{
                pointerEvents: 'none',
                userSelect: 'none',
                display: 'block',
              }}
            />
          </div>
        );
      } else if (objectUrl) {
        return (
          <div className="flex items-center justify-center h-full w-full bg-black/40 overflow-auto p-4">
            <img
              src={objectUrl}
              alt={filename}
              className="max-w-full max-h-full object-contain"
              style={{ display: 'block' }}
            />
          </div>
        );
      }
    }

    if (mimeType === PDF_MIME_TYPE && objectUrl) {
      return (
        <div className="w-full h-full">
          <iframe
            src={objectUrl}
            className="w-full h-full border-0"
            title={filename}
          />
        </div>
      );
    }

    if (TEXT_MIME_TYPES.includes(mimeType) && textContent !== null) {
      return (
        <div className="h-full overflow-auto bg-black/40 p-4">
          <pre className="text-sm text-emerald-100 whitespace-pre-wrap font-mono">
            {textContent}
          </pre>
        </div>
      );
    }

    if (VIDEO_MIME_TYPES.some((type) => mimeType.includes(type)) && objectUrl) {
      return (
        <div className="w-full h-full flex items-center justify-center bg-black/40 relative">
          {videoLoading && !videoError && (
            <div className="absolute inset-0 flex items-center justify-center bg-black/60 z-10">
              <div className="flex flex-col items-center gap-3 w-full max-w-md px-4">
                <Spinner size="md" borderColorClass="border-emerald-400" />
                <p className="text-sm text-emerald-400/80">
                  Подготовка видео...
                </p>
                {videoProgress > 0 && (
                  <div className="w-full bg-emerald-900/30 rounded-full h-1.5 overflow-hidden">
                    <div
                      className="bg-emerald-500 h-full transition-all duration-200 ease-out"
                      style={{
                        width: `${videoProgress}%`,
                        transform: 'translateZ(0)',
                      }}
                    />
                  </div>
                )}
              </div>
            </div>
          )}
          {videoError && (
            <div className="absolute inset-0 flex items-center justify-center bg-black/60 z-10">
              <div className="text-center">
                <p className="text-sm text-red-400 mb-2">
                  Ошибка загрузки видео
                </p>
                <button
                  onClick={() => {
                    setVideoError(false);
                    setVideoLoading(true);
                    setVideoProgress(0);
                    if (videoRef.current) {
                      videoRef.current.load();
                    }
                  }}
                  className="px-3 py-1.5 text-xs bg-emerald-500 hover:bg-emerald-400 text-black rounded transition-colors"
                >
                  Попробовать снова
                </button>
              </div>
            </div>
          )}
          <video
            ref={videoRef}
            src={objectUrl}
            controls
            preload="metadata"
            playsInline
            className="max-w-full max-h-full"
            style={{
              display: 'block',
              transform: 'translateZ(0)',
            }}
            onLoadedMetadata={() => {
              setVideoLoading(false);
              setVideoError(false);
            }}
            onCanPlay={() => {
              setVideoLoading(false);
            }}
            onCanPlayThrough={() => {
              setVideoLoading(false);
            }}
            onProgress={(e) => {
              const video = e.currentTarget;
              if (video.buffered.length > 0 && video.duration > 0) {
                const bufferedEnd = video.buffered.end(
                  video.buffered.length - 1
                );
                const progress = (bufferedEnd / video.duration) * 100;
                setVideoProgress(Math.min(progress, 100));
                if (progress >= 100) {
                  setVideoLoading(false);
                }
              }
            }}
            onError={() => {
              setVideoLoading(false);
              setVideoError(true);
            }}
            onLoadStart={() => {
              setVideoLoading(true);
              setVideoError(false);
              setVideoProgress(0);
            }}
            onWaiting={() => {
              setVideoLoading(true);
            }}
            onPlaying={() => {
              setVideoLoading(false);
            }}
          >
            Ваш браузер не поддерживает воспроизведение видео.
          </video>
        </div>
      );
    }

    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <p className="text-emerald-500/80 mb-2">Загрузка...</p>
        </div>
      </div>
    );
  };

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/80 backdrop-blur-sm animate-[backdropEnter_0.2s_ease-out]"
      onClick={onClose}
      style={{ willChange: 'opacity' }}
    >
      <div
        className="w-[95vw] h-[95vh] max-w-[95vw] max-h-[95vh] flex flex-col bg-black border border-emerald-700 rounded-xl overflow-hidden shadow-lg shadow-emerald-900/20 animate-[modalEnter_0.2s_ease-out]"
        onClick={(e) => e.stopPropagation()}
        style={{
          transform: 'translateZ(0)',
          willChange: 'transform, opacity',
        }}
      >
        <div className="flex items-center justify-between px-6 py-4 border-b border-emerald-700/60 bg-gradient-to-r from-black via-emerald-950/20 to-black">
          <div className="flex-1 min-w-0">
            <h3 className="text-lg font-semibold bg-gradient-to-r from-emerald-300 to-emerald-400 bg-clip-text text-transparent truncate">
              {filename}
            </h3>
            <p className="text-xs text-emerald-400/80 mt-1 font-medium">
              {mimeType}
            </p>
          </div>
          <div className="flex items-center gap-3 ml-4">
            {isProtected && (
              <span className="text-xs text-yellow-400 px-2 py-1 rounded bg-yellow-900/20 border border-yellow-700/40">
                Только просмотр
              </span>
            )}
            {!isProtected && (
              <button
                type="button"
                onClick={handleDownload}
                className="px-3 py-1.5 text-sm font-medium rounded-md bg-emerald-500 hover:bg-emerald-400 text-black transition-colors flex items-center gap-2"
              >
                <svg
                  className="w-4 h-4"
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
            <button
              type="button"
              onClick={onClose}
              className="text-emerald-400 hover:text-emerald-200 smooth-transition rounded-md p-1.5 hover:bg-emerald-900/40"
              aria-label="Закрыть"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </div>

        <div
          className="flex-1 overflow-hidden relative"
          style={
            isProtected
              ? {
                  userSelect: 'none',
                  WebkitUserSelect: 'none',
                  MozUserSelect: 'none',
                  msUserSelect: 'none',
                  WebkitTouchCallout: 'none',
                }
              : {}
          }
        >
          {renderContent()}
        </div>
      </div>
    </div>
  );
}
