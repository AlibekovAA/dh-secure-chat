import { useState } from 'react';
import { checkMediaRecorderSupport } from '../../shared/browser-support';
import { VideoRecorderModal } from './VideoRecorderModal';

type Props = {
  onRecorded: (file: File, duration: number) => void;
  onError: (error: string) => void;
  disabled?: boolean;
};

export function VideoRecorder({ onRecorded, onError, disabled }: Props) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isSupported, setIsSupported] = useState(true);

  const handleClick = () => {
    if (disabled || !isSupported) return;

    try {
      checkMediaRecorderSupport();
      setIsModalOpen(true);
    } catch (error) {
      setIsSupported(false);
      const message = error instanceof Error ? error.message : 'Запись видео не поддерживается';
      onError(message);
    }
  };

  const handleRecorded = (file: File, duration: number) => {
    setIsModalOpen(false);
    onRecorded(file, duration);
  };

  const handleCancel = () => {
    setIsModalOpen(false);
  };

  if (!isSupported) {
    return null;
  }

  return (
    <>
      <button
        type="button"
        onClick={handleClick}
        disabled={disabled}
        className="flex-shrink-0 w-10 h-10 rounded-full bg-emerald-600 hover:bg-emerald-700 disabled:bg-emerald-900/40 disabled:cursor-not-allowed text-white flex items-center justify-center transition-colors"
        title="Записать видео сообщение"
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
            d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
          />
        </svg>
      </button>

      {isModalOpen && (
        <VideoRecorderModal
          onRecorded={handleRecorded}
          onCancel={handleCancel}
          onError={onError}
        />
      )}
    </>
  );
}
