import { useState, useCallback, useEffect, useRef } from 'react';
import { AudioRecorder } from '../../shared/audio/audio-recorder';
import { checkMediaRecorderSupport } from '../../shared/browser-support';
import { MAX_VOICE_DURATION_SECONDS } from './constants';

type Props = {
  onRecorded: (file: File, duration: number) => void;
  onError: (error: string) => void;
  disabled?: boolean;
};

export function VoiceRecorder({ onRecorded, onError, disabled }: Props) {
  const [isRecording, setIsRecording] = useState(false);
  const [duration, setDuration] = useState(0);
  const [isSupported, setIsSupported] = useState(true);
  const recorderRef = useRef<AudioRecorder | null>(null);
  const durationTimerRef = useRef<number | null>(null);

  useEffect(() => {
    try {
      checkMediaRecorderSupport();
      setIsSupported(true);
    } catch (error) {
      setIsSupported(false);
      const message = error instanceof Error ? error.message : 'Запись аудио не поддерживается';
      onError(message);
    }

    return () => {
      if (recorderRef.current) {
        recorderRef.current.cleanup();
      }
      if (durationTimerRef.current) {
        clearInterval(durationTimerRef.current);
      }
    };
  }, [onError]);

  const startRecording = useCallback(async () => {
    if (!isSupported || disabled || isRecording) return;

    try {
      const recorder = new AudioRecorder({
        maxDuration: MAX_VOICE_DURATION_SECONDS * 1000,
        audioBitsPerSecond: 32000,
      });

      recorderRef.current = recorder;
      await recorder.start();
      setIsRecording(true);
      setDuration(0);

      durationTimerRef.current = window.setInterval(() => {
        const currentDuration = recorder.getDuration();
        setDuration(currentDuration);

        if (currentDuration >= MAX_VOICE_DURATION_SECONDS) {
          stopRecording();
        }
      }, 100);
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : 'Не удалось начать запись. Проверьте разрешения микрофона.';
      onError(message);
      setIsRecording(false);
    }
  }, [isSupported, disabled, isRecording, onError]);

  const stopRecording = useCallback(async () => {
    if (!recorderRef.current || !isRecording) return;

    try {
      const recorder = recorderRef.current;
      const blob = recorder.stop();
      const finalDuration = recorder.getDuration();

      if (durationTimerRef.current) {
        clearInterval(durationTimerRef.current);
        durationTimerRef.current = null;
      }

      setIsRecording(false);
      setDuration(0);

      if (blob) {
        const blobClone = blob.slice(0, blob.size, blob.type);
        const file = new File([blobClone], `voice-${finalDuration}s.webm`, {
          type: blob.type || 'audio/webm',
        });
        console.log('[VoiceRecorder] Отправка голосового сообщения:', {
          duration: finalDuration,
          filename: file.name,
          fileSize: file.size,
          blobSize: blob.size,
        });
        onRecorded(file, finalDuration);
      }

      recorder.cleanup();
      recorderRef.current = null;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Ошибка при остановке записи';
      onError(message);
      setIsRecording(false);
    }
  }, [isRecording, onRecorded, onError]);

  const cancelRecording = useCallback(() => {
    if (recorderRef.current) {
      recorderRef.current.cancel();
      recorderRef.current = null;
    }
    if (durationTimerRef.current) {
      clearInterval(durationTimerRef.current);
      durationTimerRef.current = null;
    }
    setIsRecording(false);
    setDuration(0);
  }, []);

  const formatDuration = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  if (!isSupported) {
    return null;
  }

  return (
    <div className="flex items-center gap-2">
      {isRecording ? (
        <>
          <button
            type="button"
            onClick={stopRecording}
            className="flex-shrink-0 w-10 h-10 rounded-full bg-emerald-500 hover:bg-emerald-600 text-white flex items-center justify-center transition-colors"
            title="Остановить запись"
          >
            <div className="w-4 h-4 bg-white rounded-sm" />
          </button>
          <div className="flex items-center gap-2 min-w-[80px]">
            <span className="text-xs text-emerald-400 font-mono">
              {formatDuration(duration)}
            </span>
            <div className="w-2 h-2 bg-red-500 rounded-full animate-pulse" />
          </div>
          <button
            type="button"
            onClick={cancelRecording}
            className="flex-shrink-0 px-2 py-1 text-xs text-emerald-400 hover:text-emerald-200 transition-colors"
            title="Отменить запись"
          >
            Отмена
          </button>
        </>
      ) : (
        <button
          type="button"
          onClick={startRecording}
          disabled={disabled}
          className="flex-shrink-0 w-10 h-10 rounded-full bg-emerald-600 hover:bg-emerald-700 disabled:bg-emerald-900/40 disabled:cursor-not-allowed text-white flex items-center justify-center transition-colors"
          title="Записать голосовое сообщение"
        >
          <svg
            className="w-5 h-5"
            fill="currentColor"
            viewBox="0 0 24 24"
          >
            <path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z" />
            <path d="M17 11c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z" />
          </svg>
        </button>
      )}
    </div>
  );
}
