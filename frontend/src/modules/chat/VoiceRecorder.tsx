import { useState, useCallback, useEffect, useRef } from 'react';
import { AudioRecorder } from '@/shared/audio/audio-recorder';
import { checkMediaRecorderSupport } from '@/shared/browser-support';
import { MAX_VOICE_DURATION_SECONDS } from '@/shared/constants';
import { MESSAGES } from '@/shared/messages';

type Props = {
  onRecorded: (file: File, duration: number) => void;
  onError: (error: string) => void;
  disabled?: boolean;
  onRecordingChange?: (isRecording: boolean) => void;
};

export function VoiceRecorder({
  onRecorded,
  onError,
  disabled,
  onRecordingChange,
}: Props) {
  const [isRecording, setIsRecording] = useState(false);
  const [duration, setDuration] = useState(0);
  const [isSupported, setIsSupported] = useState(true);
  const recorderRef = useRef<AudioRecorder | null>(null);
  const durationTimerRef = useRef<number | null>(null);
  const isRecordingRef = useRef(false);

  useEffect(() => {
    try {
      checkMediaRecorderSupport();
      setIsSupported(true);
    } catch (error) {
      setIsSupported(false);
      const message =
        error instanceof Error
          ? error.message
          : MESSAGES.chat.voiceRecorder.errors.notSupported;
      onError(message);
    }

    return () => {
      const currentIsRecording = isRecordingRef.current;
      if (recorderRef.current) {
        if (currentIsRecording) {
          return;
        }
        recorderRef.current.cleanup();
      }
      if (durationTimerRef.current) {
        clearInterval(durationTimerRef.current);
        durationTimerRef.current = null;
      }
    };
  }, []);

  const stopRecording = useCallback(async () => {
    if (!recorderRef.current || !isRecording) {
      return;
    }

    try {
      const recorder = recorderRef.current;
      const blob = await recorder.stop();
      const finalDuration = recorder.getDuration();

      if (durationTimerRef.current) {
        clearInterval(durationTimerRef.current);
        durationTimerRef.current = null;
      }

      isRecordingRef.current = false;
      setIsRecording(false);
      setDuration(0);
      onRecordingChange?.(false);

      if (!blob || blob.size === 0) {
        onError(MESSAGES.chat.voiceRecorder.errors.notRecorded);
        recorder.cleanup();
        recorderRef.current = null;
        return;
      }

      const blobClone = blob.slice(0, blob.size, blob.type);
      const file = new File([blobClone], `voice-${finalDuration}s.webm`, {
        type: blob.type || 'audio/webm',
      });

      if (file.size === 0) {
        onError(MESSAGES.chat.voiceRecorder.errors.emptyRecording);
        recorder.cleanup();
        recorderRef.current = null;
        return;
      }

      onRecorded(file, finalDuration);

      recorder.cleanup();
      recorderRef.current = null;
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : MESSAGES.chat.voiceRecorder.errors.stopError;
      onError(message);
      isRecordingRef.current = false;
      setIsRecording(false);
      onRecordingChange?.(false);
    }
  }, [isRecording, onRecorded, onError, onRecordingChange]);

  const startRecording = useCallback(async () => {
    if (!isSupported || disabled || isRecording) {
      return;
    }

    try {
      const recorder = new AudioRecorder({
        maxDuration: MAX_VOICE_DURATION_SECONDS * 1000,
        audioBitsPerSecond: 32000,
      });

      recorderRef.current = recorder;
      await recorder.start();
      isRecordingRef.current = true;
      setIsRecording(true);
      setDuration(0);
      onRecordingChange?.(true);

      durationTimerRef.current = window.setInterval(() => {
        const currentRecorder = recorderRef.current;
        if (!currentRecorder) {
          return;
        }
        const currentDuration = currentRecorder.getDuration();
        setDuration(currentDuration);

        if (currentDuration >= MAX_VOICE_DURATION_SECONDS) {
          stopRecording();
        }
      }, 100);
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : MESSAGES.chat.voiceRecorder.errors.startError;
      onError(message);
      isRecordingRef.current = false;
      setIsRecording(false);
      onRecordingChange?.(false);
    }
  }, [
    isSupported,
    disabled,
    isRecording,
    onError,
    stopRecording,
    onRecordingChange,
  ]);

  const cancelRecording = useCallback(() => {
    if (recorderRef.current) {
      recorderRef.current.cancel();
      recorderRef.current = null;
    }
    if (durationTimerRef.current) {
      clearInterval(durationTimerRef.current);
      durationTimerRef.current = null;
    }
    isRecordingRef.current = false;
    setIsRecording(false);
    setDuration(0);
    onRecordingChange?.(false);
  }, [onRecordingChange]);

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
            title={MESSAGES.chat.voiceRecorder.titles.stop}
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
            title={MESSAGES.chat.voiceRecorder.titles.cancel}
          >
            {MESSAGES.chat.voiceRecorder.labels.cancel}
          </button>
        </>
      ) : (
        <button
          type="button"
          onClick={startRecording}
          disabled={disabled}
          className="flex-shrink-0 w-10 h-10 rounded-full bg-emerald-600 hover:bg-emerald-700 disabled:bg-emerald-900/40 disabled:cursor-not-allowed text-white flex items-center justify-center transition-colors"
          title={MESSAGES.chat.voiceRecorder.titles.record}
        >
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z" />
            <path d="M17 11c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z" />
          </svg>
        </button>
      )}
    </div>
  );
}
