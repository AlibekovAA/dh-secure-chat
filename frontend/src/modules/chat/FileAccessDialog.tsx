import { useRef, useState } from 'react';
import { MESSAGES } from '@/shared/messages';
import type { FileAccessMode } from '@/shared/websocket/types';
import { useDialogFocus } from '@/shared/ui/useDialogFocus';

type Props = {
  filename: string;
  onSelect: (mode: FileAccessMode) => void;
  onCancel: () => void;
};

export function FileAccessDialog({ filename, onSelect, onCancel }: Props) {
  const [selectedMode, setSelectedMode] = useState<FileAccessMode>('both');
  const dialogRef = useRef<HTMLDivElement>(null);

  useDialogFocus(dialogRef, onCancel);

  const handleConfirm = () => {
    onSelect(selectedMode);
  };

  const options = [
    {
      value: 'both' as FileAccessMode,
      title: MESSAGES.chat.fileAccessDialog.modes.both.title,
      description: MESSAGES.chat.fileAccessDialog.modes.both.description,
    },
    {
      value: 'view_only' as FileAccessMode,
      title: MESSAGES.chat.fileAccessDialog.modes.viewOnly.title,
      description: MESSAGES.chat.fileAccessDialog.modes.viewOnly.description,
    },
    {
      value: 'download_only' as FileAccessMode,
      title: MESSAGES.chat.fileAccessDialog.modes.downloadOnly.title,
      description:
        MESSAGES.chat.fileAccessDialog.modes.downloadOnly.description,
    },
  ];

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm animate-[fadeIn_0.2s_ease-out]"
      onClick={onCancel}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="file-access-dialog-title"
        tabIndex={-1}
        className="w-full max-w-md mx-4 bg-[var(--color-surface)] border border-slate-700/80 rounded-lg overflow-hidden animate-[scaleIn_0.2s_ease-out]"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-6 py-4 border-b border-slate-700/80 border-l-2 border-l-emerald-500/70 bg-[var(--color-surface-raised)]">
          <h3
            id="file-access-dialog-title"
            className="text-lg font-semibold text-slate-100"
          >
            {MESSAGES.chat.fileAccessDialog.title}
          </h3>
          <p className="text-sm text-emerald-400/90 mt-1 truncate font-medium">
            {filename}
          </p>
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
                  ${
                    isSelected
                      ? 'border-emerald-500/70 bg-emerald-500/10'
                      : 'border-slate-700/70 bg-slate-900/40 hover:bg-slate-800/60 hover:border-slate-600'
                  }
                `}
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
                      ${
                        isSelected
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
                    <div
                      className="absolute inset-0 rounded-full border-2 border-emerald-400/50 animate-ping"
                      style={{ animationDuration: '1.5s' }}
                    />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div
                    className={`text-sm font-semibold transition-colors duration-200 mb-0.5 ${
                      isSelected ? 'text-emerald-200' : 'text-emerald-100'
                    }`}
                  >
                    {option.title}
                  </div>
                  <div
                    className={`text-xs transition-colors duration-200 ${
                      isSelected ? 'text-emerald-400/90' : 'text-emerald-500/70'
                    }`}
                  >
                    {option.description}
                  </div>
                </div>
              </label>
            );
          })}
        </div>

        <div className="px-6 py-4 border-t border-slate-700/80 bg-[var(--color-surface-raised)] flex items-center justify-end gap-3">
          <button
            type="button"
            onClick={onCancel}
            className="px-4 py-2 text-sm font-medium rounded-md bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-colors duration-200"
          >
            {MESSAGES.chat.fileAccessDialog.actions.cancel}
          </button>
          <button
            type="button"
            onClick={handleConfirm}
            className="px-4 py-2 text-sm font-semibold rounded-md bg-emerald-500 hover:bg-emerald-400 text-slate-950 transition-colors duration-200"
          >
            {MESSAGES.chat.fileAccessDialog.actions.send}
          </button>
        </div>
      </div>
    </div>
  );
}
