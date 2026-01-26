import { useState, useRef, useEffect, useCallback } from 'react';
import { checkMediaRecorderSupport } from '@/shared/browser-support';
import {
  MAX_FILE_SIZE,
  VIDEO_RECORDER_CHECK_INTERVAL_MS,
  VIDEO_RECORDER_DURATION_UPDATE_DELAY_MS,
  VIDEO_RECORDER_DURATION_UPDATE_INTERVAL_MS,
  VIDEO_RECORDER_TIMESLICE_MS,
  MS_PER_SECOND,
  BYTES_PER_MB,
} from '@/shared/constants';

type Props = {
  onRecorded: (file: File, duration: number) => void;
  onCancel: () => void;
  onError: (error: string) => void;
};

export function VideoRecorderModal({ onRecorded, onCancel, onError }: Props) {
  const [isRecording, setIsRecording] = useState(false);
  const [duration, setDuration] = useState(0);
  const [isInitializing, setIsInitializing] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const startTimeRef = useRef<number>(0);
  const durationTimerRef = useRef<number | null>(null);
  const isCancelledRef = useRef<boolean>(false);

  const stopStream = useCallback(() => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
    }
  }, []);

  const stopDurationTimer = useCallback(() => {
    if (durationTimerRef.current) {
      clearInterval(durationTimerRef.current);
      durationTimerRef.current = null;
    }
  }, []);

  const cleanup = useCallback(() => {
    stopDurationTimer();
    if (
      mediaRecorderRef.current &&
      mediaRecorderRef.current.state !== 'inactive'
    ) {
      try {
        mediaRecorderRef.current.stop();
      } catch {
        void 0;
      }
    }
    mediaRecorderRef.current = null;
    chunksRef.current = [];
    stopStream();
  }, [stopDurationTimer, stopStream]);

  useEffect(() => {
    const initCamera = async () => {
      try {
        checkMediaRecorderSupport();
        setIsInitializing(true);
        setError(null);

        const stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: 'user' },
          audio: true,
        });

        streamRef.current = stream;

        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          videoRef.current.muted = true;
          videoRef.current.autoplay = true;
          videoRef.current.playsInline = true;
          try {
            await videoRef.current.play();
          } catch (_playError) {
            setTimeout(async () => {
              if (videoRef.current && videoRef.current.paused) {
                try {
                  await videoRef.current.play();
                } catch {
                  void 0;
                }
              }
            }, VIDEO_RECORDER_DURATION_UPDATE_DELAY_MS);
          }
        }

        setIsInitializing(false);
      } catch (err) {
        const message =
          err instanceof Error
            ? err.message
            : 'Не удалось получить доступ к камере';
        setError(message);
        onError(message);
        setIsInitializing(false);
      }
    };

    initCamera();

    return () => {
      cleanup();
    };
  }, [onError, cleanup]);

  useEffect(() => {
    if (!isRecording) return;

    const checkVideo = () => {
      if (videoRef.current && streamRef.current) {
        if (!videoRef.current.srcObject) {
          videoRef.current.srcObject = streamRef.current;
        }
        if (videoRef.current.paused) {
          videoRef.current.play().catch(() => {});
        }
      }
    };

    const interval = setInterval(checkVideo, VIDEO_RECORDER_CHECK_INTERVAL_MS);

    return () => {
      clearInterval(interval);
    };
  }, [isRecording]);

  const startRecording = useCallback(async () => {
    if (!streamRef.current || isRecording) return;

    try {
      if (videoRef.current && !videoRef.current.srcObject) {
        videoRef.current.srcObject = streamRef.current;
        await videoRef.current.play();
      }

      const mimeTypes = [
        'video/webm;codecs=vp9,opus',
        'video/webm;codecs=vp8,opus',
        'video/webm',
        'video/mp4',
      ];

      let selectedMimeType = 'video/webm';
      for (const mimeType of mimeTypes) {
        if (MediaRecorder.isTypeSupported(mimeType)) {
          selectedMimeType = mimeType;
          break;
        }
      }

      const mediaRecorder = new MediaRecorder(streamRef.current, {
        mimeType: selectedMimeType,
        videoBitsPerSecond: 2500000,
      });

      mediaRecorderRef.current = mediaRecorder;
      chunksRef.current = [];

      mediaRecorder.ondataavailable = (event) => {
        if (event.data && event.data.size > 0) {
          chunksRef.current.push(event.data);
        }
      };

      mediaRecorder.onerror = (event) => {
        const errorEvent = event as ErrorEvent;
        const message = errorEvent.message || 'Ошибка при записи видео';
        setError(message);
        onError(message);
        cleanup();
        setIsRecording(false);
      };

      mediaRecorder.onstop = () => {
        stopDurationTimer();

        if (isCancelledRef.current) {
          isCancelledRef.current = false;
          cleanup();
          setIsRecording(false);
          return;
        }

        const blob = new Blob(chunksRef.current, { type: selectedMimeType });
        const finalDuration = Math.floor(
          (Date.now() - startTimeRef.current) / MS_PER_SECOND
        );

        if (blob.size === 0 || finalDuration === 0) {
          cleanup();
          setIsRecording(false);
          return;
        }

        if (blob.size > MAX_FILE_SIZE) {
          const message = `Видео слишком большое (${(blob.size / BYTES_PER_MB).toFixed(1)} MB). Максимальный размер: ${(MAX_FILE_SIZE / BYTES_PER_MB).toFixed(0)} MB`;
          setError(message);
          onError(message);
          cleanup();
          setIsRecording(false);
          return;
        }

        const extension = selectedMimeType.includes('mp4') ? 'mp4' : 'webm';
        const file = new File([blob], `video-${finalDuration}s.${extension}`, {
          type: selectedMimeType,
        });

        cleanup();
        setIsRecording(false);
        onRecorded(file, finalDuration);
      };

      mediaRecorder.start(VIDEO_RECORDER_TIMESLICE_MS);
      setIsRecording(true);
      setDuration(0);
      startTimeRef.current = Date.now();
      isCancelledRef.current = false;

      if (videoRef.current && videoRef.current.paused) {
        videoRef.current.play().catch(() => {});
      }

      if (videoRef.current && !videoRef.current.srcObject) {
        videoRef.current.srcObject = streamRef.current;
        videoRef.current.play().catch(() => {});
      }

      durationTimerRef.current = window.setInterval(() => {
        const currentDuration = Math.floor(
          (Date.now() - startTimeRef.current) / MS_PER_SECOND
        );
        setDuration(currentDuration);
      }, VIDEO_RECORDER_DURATION_UPDATE_INTERVAL_MS);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Не удалось начать запись видео';
      setError(message);
      onError(message);
      setIsRecording(false);
    }
  }, [isRecording, onError, onRecorded, cleanup, stopDurationTimer]);

  const stopRecording = useCallback(() => {
    if (
      mediaRecorderRef.current &&
      mediaRecorderRef.current.state === 'recording'
    ) {
      mediaRecorderRef.current.stop();
    }
  }, []);

  const cancelRecording = useCallback(() => {
    isCancelledRef.current = true;
    if (mediaRecorderRef.current) {
      if (mediaRecorderRef.current.state === 'recording') {
        mediaRecorderRef.current.stop();
      }
      chunksRef.current = [];
    }
    cleanup();
    setIsRecording(false);
    setDuration(0);
    onCancel();
  }, [onCancel, cleanup]);

  const formatDuration = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm animate-[fadeIn_0.2s_ease-out]"
      style={{ willChange: 'opacity' }}
    >
      <div
        className="relative w-full max-w-2xl mx-4 bg-emerald-950/95 border border-emerald-700/50 rounded-lg shadow-2xl overflow-hidden animate-[scaleIn_0.2s_ease-out]"
        style={{ willChange: 'transform, opacity' }}
      >
        <div className="relative aspect-video bg-black">
          {isInitializing ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <div className="w-12 h-12 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin mx-auto mb-4" />
                <p className="text-emerald-400 text-sm">
                  Инициализация камеры...
                </p>
              </div>
            </div>
          ) : error ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <p className="text-red-400 text-sm mb-4">{error}</p>
                <button
                  type="button"
                  onClick={onCancel}
                  className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors"
                >
                  Закрыть
                </button>
              </div>
            </div>
          ) : (
            <>
              <video
                ref={videoRef}
                autoPlay
                playsInline
                muted
                className="w-full h-full object-cover"
                style={{ transform: 'scaleX(-1)' }}
                onLoadedMetadata={() => {
                  if (videoRef.current) {
                    if (!videoRef.current.srcObject && streamRef.current) {
                      videoRef.current.srcObject = streamRef.current;
                    }
                    if (videoRef.current.paused) {
                      videoRef.current.play().catch(() => {});
                    }
                  }
                }}
                onCanPlay={() => {
                  if (videoRef.current) {
                    if (!videoRef.current.srcObject && streamRef.current) {
                      videoRef.current.srcObject = streamRef.current;
                    }
                    if (videoRef.current.paused) {
                      videoRef.current.play().catch(() => {});
                    }
                  }
                }}
                onPause={() => {
                  if (videoRef.current && streamRef.current && !isRecording) {
                    videoRef.current.play().catch(() => {});
                  }
                }}
              />
              {isRecording && (
                <div className="absolute top-4 left-4 flex items-center gap-2 bg-black/60 px-3 py-1.5 rounded-lg">
                  <div className="w-2 h-2 bg-red-500 rounded-full animate-pulse" />
                  <span className="text-white text-sm font-mono">
                    {formatDuration(duration)}
                  </span>
                </div>
              )}
            </>
          )}
        </div>

        <div className="p-4 bg-emerald-950/95 border-t border-emerald-700/50">
          <div className="flex items-center justify-center gap-4">
            {!isInitializing && !error && (
              <>
                {!isRecording ? (
                  <>
                    <button
                      type="button"
                      onClick={startRecording}
                      className="flex items-center justify-center w-14 h-14 rounded-full bg-emerald-600 hover:bg-emerald-700 text-white transition-colors shadow-lg"
                      title="Начать запись"
                    >
                      <svg
                        className="w-6 h-6"
                        fill="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path d="M8 5v14l11-7z" />
                      </svg>
                    </button>
                    <button
                      type="button"
                      onClick={onCancel}
                      className="px-4 py-2 text-emerald-400 hover:text-emerald-200 transition-colors"
                      title="Отменить"
                    >
                      Отменить
                    </button>
                  </>
                ) : (
                  <>
                    <button
                      type="button"
                      onClick={stopRecording}
                      className="flex items-center justify-center w-14 h-14 rounded-full bg-red-600 hover:bg-red-700 text-white transition-colors shadow-lg"
                      title="Остановить запись"
                    >
                      <div className="w-5 h-5 bg-white rounded-sm" />
                    </button>
                    <button
                      type="button"
                      onClick={cancelRecording}
                      className="px-4 py-2 text-emerald-400 hover:text-emerald-200 transition-colors"
                      title="Отменить запись"
                    >
                      Отменить
                    </button>
                  </>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
