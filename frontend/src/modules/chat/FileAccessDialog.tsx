import { useState } from 'react';

export type FileAccessMode = 'download_only' | 'view_only' | 'both';

type Props = {
  filename: string;
  onSelect: (mode: FileAccessMode) => void;
  onCancel: () => void;
};

export function FileAccessDialog({ filename, onSelect, onCancel }: Props) {
  const [selectedMode, setSelectedMode] = useState<FileAccessMode>('both');

  const handleConfirm = () => {
    onSelect(selectedMode);
  };

  const options = [
    {
      value: 'both' as FileAccessMode,
      title: 'Скачивание и просмотр',
      description: 'Собеседник сможет скачать и просмотреть файл',
    },
    {
      value: 'view_only' as FileAccessMode,
      title: 'Только просмотр',
      description: 'Собеседник сможет только просмотреть файл, но не скачать',
    },
    {
      value: 'download_only' as FileAccessMode,
      title: 'Только скачивание',
      description: 'Собеседник сможет только скачать файл, но не просмотреть',
    },
  ];

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm animate-[fadeIn_0.2s_ease-out]"
      onClick={onCancel}
      style={{ willChange: 'opacity' }}
    >
      <div
        className="w-full max-w-md mx-4 bg-black border border-emerald-700 rounded-xl overflow-hidden animate-[scaleIn_0.2s_ease-out] shadow-2xl shadow-emerald-900/30"
        onClick={(e) => e.stopPropagation()}
        style={{ willChange: 'transform, opacity' }}
      >
        <div className="px-6 py-4 border-b border-emerald-700/60 bg-gradient-to-r from-black via-emerald-950/20 to-black">
          <h3 className="text-lg font-semibold bg-gradient-to-r from-emerald-300 to-emerald-400 bg-clip-text text-transparent">
            Выберите режим доступа
          </h3>
          <p className="text-sm text-emerald-400/90 mt-1 truncate font-medium">{filename}</p>
        </div>

        <div className="px-6 py-4 space-y-3">
          {options.map((option) => {
            const isSelected = selectedMode === option.value;
            return (
              <label
                key={option.value}
                className={`
                  flex items-start gap-3 p-3.5 rounded-lg border cursor-pointer
                  transition-all duration-200 ease-out
                  ${isSelected
                    ? 'border-emerald-500/80 bg-gradient-to-r from-emerald-500/20 via-emerald-500/15 to-emerald-500/20 shadow-lg shadow-emerald-500/20 scale-[1.02]'
                    : 'border-emerald-700/40 bg-emerald-900/10 hover:bg-emerald-900/20 hover:border-emerald-700/60 hover:scale-[1.01]'
                  }
                `}
                style={{ willChange: 'transform, background-color, border-color' }}
              >
                <div className="relative flex-shrink-0 mt-0.5">
                  <input
                    type="radio"
                    name="accessMode"
                    value={option.value}
                    checked={isSelected}
                    onChange={() => setSelectedMode(option.value)}
                    className="sr-only"
                  />
                  <div
                    className={`
                      w-5 h-5 rounded-full border-2 flex items-center justify-center
                      transition-all duration-200 ease-out
                      ${isSelected
                        ? 'border-emerald-400 bg-emerald-500/20'
                        : 'border-emerald-600 bg-transparent'
                      }
                    `}
                  >
                    {isSelected && (
                      <div className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-[scaleIn_0.15s_ease-out]" />
                    )}
                  </div>
                  {isSelected && (
                    <div className="absolute inset-0 rounded-full border-2 border-emerald-400/50 animate-ping" style={{ animationDuration: '1.5s' }} />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className={`text-sm font-semibold transition-colors duration-200 mb-0.5 ${
                    isSelected ? 'text-emerald-200' : 'text-emerald-100'
                  }`}>
                    {option.title}
                  </div>
                  <div className={`text-xs transition-colors duration-200 ${
                    isSelected ? 'text-emerald-400/90' : 'text-emerald-500/70'
                  }`}>
                    {option.description}
                  </div>
                </div>
              </label>
            );
          })}
        </div>

        <div className="px-6 py-4 border-t border-emerald-700/60 bg-black/80 flex items-center justify-end gap-3">
          <button
            type="button"
            onClick={onCancel}
            className="px-4 py-2 text-sm font-medium rounded-md bg-emerald-900/40 hover:bg-emerald-900/60 text-emerald-300 border border-emerald-700/60 transition-all duration-200 hover:scale-105 active:scale-95"
            style={{ willChange: 'transform' }}
          >
            Отмена
          </button>
          <button
            type="button"
            onClick={handleConfirm}
            className="px-4 py-2 text-sm font-medium rounded-md bg-gradient-to-r from-emerald-500 to-emerald-400 hover:from-emerald-400 hover:to-emerald-300 text-black transition-all duration-200 hover:scale-105 active:scale-95 shadow-lg shadow-emerald-500/30"
            style={{ willChange: 'transform' }}
          >
            Отправить
          </button>
        </div>
      </div>
    </div>
  );
}
