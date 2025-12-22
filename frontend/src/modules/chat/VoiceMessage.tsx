import { useState, useRef, useEffect } from 'react';

type Props = {
  duration: number;
  blob?: Blob;
  isOwn: boolean;
  onPlaybackChange?: (active: boolean) => void;
};

function formatDuration(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

export function VoiceMessage({ duration, blob, isOwn, onPlaybackChange }: Props) {
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [metadataDuration, setMetadataDuration] = useState<number | null>(null);
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);

  useEffect(() => {
    if (!blob) {
      if (audioUrl) {
        if (audioRef.current) {
          audioRef.current.pause();
          audioRef.current.src = '';
          audioRef.current.load();
        }
        URL.revokeObjectURL(audioUrl);
        setAudioUrl(null);
      }
      setCurrentTime(0);
      setIsPlaying(false);
      setMetadataDuration(null);
      return;
    }

    if (blob.size === 0) {
      return;
    }

    if (!blob.type || !blob.type.startsWith('audio/')) {
      return;
    }

    setCurrentTime(0);
    setIsPlaying(false);

    let url: string | null = null;
    try {
      url = URL.createObjectURL(blob);
      setAudioUrl(url);
    } catch {
      return;
    }

    return () => {
      if (audioRef.current) {
        audioRef.current.pause();
        audioRef.current.src = '';
        audioRef.current.load();
      }
      if (url) {
        URL.revokeObjectURL(url);
      }
    };
  }, [blob, duration]);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio || !audioUrl) {
      if (audio && !audioUrl) {
        audio.pause();
        audio.src = '';
        audio.load();
      }
      return;
    }

    if (audio.src !== audioUrl) {
      try {
        audio.src = audioUrl;
        audio.load();
        audio.currentTime = 0.01;
        audio.currentTime = 0;
      } catch (error) {
        setIsLoading(false);
        return;
      }
    } else if (audio.readyState === 0) {
      audio.load();
    }

    let loadTimeout: number | null = null;
    let metadataCheckInterval: number | null = null;

    const updateTime = () => {
      const current = audio.currentTime;
      setCurrentTime(current);
      const dur = audio.duration;
      if (dur && !isNaN(dur) && dur > 0 && dur !== Infinity) {
        const newDuration = Math.floor(dur);
        setMetadataDuration(newDuration);
      }
    };

    const checkMetadata = () => {
      const dur = audio.duration;
      if (dur && !isNaN(dur) && dur > 0 && dur !== Infinity) {
        const newDuration = Math.floor(dur);
        setMetadataDuration(newDuration);
        if (metadataCheckInterval) {
          clearInterval(metadataCheckInterval);
          metadataCheckInterval = null;
        }
      }
    };
    const handlePlay = () => {
      setIsPlaying(true);
      onPlaybackChange?.(true);
    };
    const handlePause = () => {
      setIsPlaying(false);
      onPlaybackChange?.(false);
    };
    const handleEnded = () => {
      setIsPlaying(false);
      setCurrentTime(0);
      onPlaybackChange?.(false);
    };
    const handleLoadStart = () => {
      setIsLoading(true);
      loadTimeout = window.setTimeout(() => {
        setIsLoading(false);
      }, 5000);
    };
    const handleLoadedMetadata = () => {
      checkMetadata();
      if (loadTimeout) {
        clearTimeout(loadTimeout);
        loadTimeout = null;
      }
      if (metadataCheckInterval) {
        clearInterval(metadataCheckInterval);
        metadataCheckInterval = null;
      }
      setIsLoading(false);
    };
    const handleCanPlay = () => {
      checkMetadata();
      if (loadTimeout) {
        clearTimeout(loadTimeout);
        loadTimeout = null;
      }
      if (metadataCheckInterval) {
        clearInterval(metadataCheckInterval);
        metadataCheckInterval = null;
      }
      setIsLoading(false);
    };
    const handleError = (e: Event) => {
      if (loadTimeout) {
        clearTimeout(loadTimeout);
        loadTimeout = null;
      }
      setIsLoading(false);
    };

    audio.addEventListener('timeupdate', updateTime);
    audio.addEventListener('play', handlePlay);
    audio.addEventListener('pause', handlePause);
    audio.addEventListener('ended', handleEnded);
    audio.addEventListener('loadstart', handleLoadStart);
    audio.addEventListener('loadedmetadata', handleLoadedMetadata);
    audio.addEventListener('canplay', handleCanPlay);
    audio.addEventListener('canplaythrough', handleCanPlay);
    audio.addEventListener('error', handleError);

    const tryLoadMetadata = () => {
      if (audio.readyState >= 2) {
        checkMetadata();
        setIsLoading(false);
        if (metadataCheckInterval) {
          clearInterval(metadataCheckInterval);
          metadataCheckInterval = null;
        }
      } else {
        audio.load();
      }
    };

    tryLoadMetadata();
    if (!metadataCheckInterval) {
      metadataCheckInterval = window.setInterval(() => {
        checkMetadata();
        if (audio.readyState >= 2) {
          if (metadataCheckInterval) {
            clearInterval(metadataCheckInterval);
            metadataCheckInterval = null;
          }
        }
      }, 100);
    }

    return () => {
      if (loadTimeout) {
        clearTimeout(loadTimeout);
      }
      if (metadataCheckInterval) {
        clearInterval(metadataCheckInterval);
      }
      audio.removeEventListener('timeupdate', updateTime);
      audio.removeEventListener('play', handlePlay);
      audio.removeEventListener('pause', handlePause);
      audio.removeEventListener('ended', handleEnded);
      audio.removeEventListener('loadstart', handleLoadStart);
      audio.removeEventListener('loadedmetadata', handleLoadedMetadata);
      audio.removeEventListener('canplay', handleCanPlay);
      audio.removeEventListener('canplaythrough', handleCanPlay);
      audio.removeEventListener('error', handleError);
    };
  }, [audioUrl, blob]);

  const handlePlayPause = () => {
    const audio = audioRef.current;
    if (!audio || !audioUrl) return;

    if (isPlaying) {
      audio.pause();
    } else {
      audio.play().catch(() => {
        setIsPlaying(false);
      });
    }
  };

  const displayDuration = metadataDuration !== null && metadataDuration > 0 ? metadataDuration : duration;
  const progress = displayDuration > 0 ? (currentTime / displayDuration) * 100 : 0;

  return (
    <div
      className={`flex items-center gap-3 p-3 rounded-lg border ${
        isOwn
          ? 'bg-emerald-500/20 border-emerald-500/40'
          : 'bg-emerald-900/20 border-emerald-700/40'
      }`}
    >
      <button
        type="button"
        onClick={handlePlayPause}
        disabled={!blob || isLoading}
        className={`flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center transition-colors ${
          isOwn
            ? 'bg-emerald-500/30 hover:bg-emerald-500/40 text-emerald-50'
            : 'bg-emerald-700/30 hover:bg-emerald-700/40 text-emerald-100'
        } disabled:opacity-50 disabled:cursor-not-allowed`}
      >
        {isLoading ? (
          <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
        ) : isPlaying ? (
          <svg
            className="w-5 h-5"
            fill="currentColor"
            viewBox="0 0 24 24"
          >
            <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
          </svg>
        ) : (
          <svg
            className="w-5 h-5 ml-0.5"
            fill="currentColor"
            viewBox="0 0 24 24"
          >
            <path d="M8 5v14l11-7z" />
          </svg>
        )}
      </button>

      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-1">
          <span
            className={`text-xs font-medium ${
              isOwn ? 'text-emerald-50' : 'text-emerald-100'
            }`}
          >
            {formatDuration(Math.floor(currentTime || 0))} / {formatDuration(displayDuration)}
          </span>
        </div>

        <div className="relative h-1.5 bg-emerald-900/40 rounded-full overflow-hidden">
          <div
            className={`absolute top-0 left-0 h-full transition-all duration-100 ${
              isOwn ? 'bg-emerald-400' : 'bg-emerald-500'
            }`}
            style={{ width: `${progress}%` }}
          />
        </div>
      </div>

      <audio
        ref={audioRef}
        preload="metadata"
        crossOrigin="anonymous"
        onLoadedMetadata={(e) => {
          const audio = e.currentTarget;
          const dur = audio.duration;
          if (dur && !isNaN(dur) && dur > 0 && dur !== Infinity) {
            const newDuration = Math.floor(dur);
            setMetadataDuration(newDuration);
          }
        }}
        onError={() => {
          setIsLoading(false);
        }}
      />
    </div>
  );
}
