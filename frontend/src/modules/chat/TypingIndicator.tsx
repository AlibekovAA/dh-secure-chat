import { memo } from 'react';

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
      style={{ willChange: 'opacity' }}
    >
      <div className="max-w-[75%] rounded-lg px-4 py-2.5 bg-gradient-to-r from-emerald-900/20 via-emerald-900/15 to-emerald-900/20 border border-emerald-700/40 smooth-transition shadow-sm">
        <div className="flex items-center gap-2">
          <div className="flex gap-1.5 px-1">
            <div
              className="w-2 h-2 rounded-full bg-gradient-to-br from-emerald-400 to-emerald-500 animate-typing shadow-sm shadow-emerald-400/30"
              style={{ animationDelay: '0ms', willChange: 'transform' }}
            />
            <div
              className="w-2 h-2 rounded-full bg-gradient-to-br from-emerald-400 to-emerald-500 animate-typing shadow-sm shadow-emerald-400/30"
              style={{ animationDelay: '150ms', willChange: 'transform' }}
            />
            <div
              className="w-2 h-2 rounded-full bg-gradient-to-br from-emerald-400 to-emerald-500 animate-typing shadow-sm shadow-emerald-400/30"
              style={{ animationDelay: '300ms', willChange: 'transform' }}
            />
          </div>
          <span className="text-xs text-emerald-300/90 italic leading-relaxed font-medium">печатает...</span>
        </div>
      </div>
    </div>
  );
}

export const TypingIndicator = memo(TypingIndicatorComponent);
