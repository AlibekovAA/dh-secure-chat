import { useEffect, useState, useRef, useCallback } from 'react';
import { generateVideoThumbnail } from '@/modules/chat/video-thumbnail';
import {
  VIDEO_THUMBNAIL_INTERSECTION_THRESHOLD,
  VIDEO_THUMBNAIL_REQUEST_IDLE_TIMEOUT_MS,
  VIDEO_THUMBNAIL_ROOT_MARGIN,
} from '@/shared/constants';
import { Spinner } from '@/shared/ui/Spinner';

type Props = {
  blob: Blob;
  filename: string;
  fileId: string;
  onClick?: () => void;
  isOwn: boolean;
};

export function VideoCircle({ blob, filename, fileId, onClick, isOwn }: Props) {
  const [thumbnail, setThumbnail] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [videoUrl, setVideoUrl] = useState<string | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const videoRef = useRef<HTMLVideoElement>(null);

  const loadThumbnail = useCallback(async () => {
    if (!blob || thumbnail) return;

    const generateThumbnail = async () => {
      try {
        setIsLoading(true);
        setError(false);
        const dataUrl = await generateVideoThumbnail(blob, fileId);
        setThumbnail(dataUrl);
      } catch (err) {
        setError(true);
      } finally {
        setIsLoading(false);
      }
    };

    if ('requestIdleCallback' in window) {
      requestIdleCallback(generateThumbnail, {
        timeout: VIDEO_THUMBNAIL_REQUEST_IDLE_TIMEOUT_MS,
      });
    } else {
      setTimeout(generateThumbnail, 0);
    }
  }, [blob, fileId, thumbnail]);

  useEffect(() => {
    if (!blob || blob.size === 0) return;

    let url: string | null = null;
    try {
      url = URL.createObjectURL(blob);
      setVideoUrl(url);
    } catch (_err) {
      void _err;
    }

    return () => {
      if (url) {
        URL.revokeObjectURL(url);
      }
    };
  }, [blob]);

  useEffect(() => {
    if (!containerRef.current) return;

    observerRef.current = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting && !thumbnail && !error) {
            loadThumbnail();
          }
        });
      },
      {
        rootMargin: VIDEO_THUMBNAIL_ROOT_MARGIN,
        threshold: VIDEO_THUMBNAIL_INTERSECTION_THRESHOLD,
      }
    );

    observerRef.current.observe(containerRef.current);

    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect();
      }
    };
  }, [loadThumbnail, thumbnail, error]);

  const handlePlayClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      e.preventDefault();
      if (videoRef.current && videoUrl) {
        if (videoRef.current.paused) {
          videoRef.current
            .play()
            .then(() => {
              setIsPlaying(true);
            })
            .catch((_err) => {
              if (onClick) {
                onClick();
              }
            });
        } else {
          videoRef.current.pause();
          setIsPlaying(false);
        }
      } else if (onClick) {
        onClick();
      }
    },
    [onClick, videoUrl]
  );

  const handleVideoEnded = useCallback(() => {
    setIsPlaying(false);
    if (videoRef.current) {
      videoRef.current.currentTime = 0;
    }
  }, []);

  const renderPlayButton = () => (
    <div className="absolute inset-0 flex items-center justify-center bg-black/30 hover:bg-black/20 transition-colors">
      <div className="w-12 h-12 rounded-full bg-emerald-500/90 hover:bg-emerald-400 flex items-center justify-center shadow-lg shadow-emerald-900/50 transition-colors">
        <svg
          className="w-6 h-6 text-black ml-1"
          fill="currentColor"
          viewBox="0 0 24 24"
        >
          <path d="M8 5v14l11-7z" />
        </svg>
      </div>
    </div>
  );

  return (
    <div
      ref={containerRef}
      className="flex items-center justify-center"
      style={{ willChange: 'transform' }}
    >
      <button
        type="button"
        onClick={handlePlayClick}
        className={`relative w-32 h-32 rounded-full overflow-hidden border-2 transition-all duration-200 ease-out ${
          isOwn
            ? 'border-emerald-500/60 hover:border-emerald-400/80'
            : 'border-emerald-700/60 hover:border-emerald-600/80'
        } ${isLoading ? 'opacity-70' : 'opacity-100'} hover:scale-105 active:scale-95`}
        style={{ transform: 'translateZ(0)' }}
        aria-label={`Воспроизвести видео: ${filename}`}
      >
        {videoUrl ? (
          <>
            <video
              ref={videoRef}
              src={videoUrl}
              className={`w-full h-full object-cover rounded-full ${isPlaying ? 'block' : 'hidden'}`}
              playsInline
              loop={false}
              onEnded={handleVideoEnded}
              onPause={() => setIsPlaying(false)}
              onPlay={() => setIsPlaying(true)}
              style={{ objectPosition: 'center' }}
            />
            {!isPlaying && (
              <>
                {thumbnail && !error ? (
                  <img
                    src={thumbnail}
                    alt=""
                    className="w-full h-full object-cover rounded-full"
                    draggable={false}
                    style={{ objectPosition: 'center' }}
                  />
                ) : (
                  <div className="w-full h-full bg-emerald-900/20" />
                )}
                {renderPlayButton()}
              </>
            )}
          </>
        ) : thumbnail && !error ? (
          <>
            <img
              src={thumbnail}
              alt=""
              className="w-full h-full object-cover rounded-full"
              draggable={false}
              style={{ objectPosition: 'center' }}
            />
            {renderPlayButton()}
          </>
        ) : isLoading ? (
          <div className="w-full h-full flex items-center justify-center bg-emerald-900/20">
            <Spinner size="md" borderColorClass="border-emerald-400" />
          </div>
        ) : (
          <div className="w-full h-full flex items-center justify-center bg-emerald-900/20">
            <svg
              className="w-12 h-12 text-emerald-500/60"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
              />
            </svg>
          </div>
        )}
      </button>
    </div>
  );
}
