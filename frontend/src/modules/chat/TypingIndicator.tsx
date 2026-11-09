import { memo } from 'react';
import { MESSAGES } from '@/shared/messages';

type Props = {
  isVisible: boolean;
};

function TypingIndicatorComponent({ isVisible }: Props) {
  if (!isVisible) {
    return null;
  }

  return (
    <div
      className="flex justify-start animate-[fadeIn_0.2s_ease-out]"
      role="status"
    >
      <div className="max-w-[75%] rounded-lg px-4 py-2.5 bg-[var(--color-surface-raised)] border border-slate-700/70 smooth-transition">
        <div className="flex items-center gap-2">
          <div className="flex gap-1.5 px-1">
            <div
              className="w-2 h-2 rounded-full bg-emerald-400 animate-typing"
              style={{ animationDelay: '0ms' }}
            />
            <div
              className="w-2 h-2 rounded-full bg-emerald-400 animate-typing"
              style={{ animationDelay: '150ms' }}
            />
            <div
              className="w-2 h-2 rounded-full bg-emerald-400 animate-typing"
              style={{ animationDelay: '300ms' }}
            />
          </div>
          <span className="text-xs text-emerald-300/90 italic leading-relaxed font-medium">
            {MESSAGES.chat.typingIndicator.label}
          </span>
        </div>
      </div>
    </div>
  );
}

export const TypingIndicator = memo(TypingIndicatorComponent);
