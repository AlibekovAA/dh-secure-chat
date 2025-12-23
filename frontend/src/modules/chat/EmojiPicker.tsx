import { useRef, useLayoutEffect, useState, useMemo, useCallback } from 'react';

const EMOJI_LIST = [
  '👍',
  '❤️',
  '😂',
  '😮',
  '😢',
  '🙏',
  '🔥',
  '👏',
  '🎉',
  '💯',
  '😍',
  '🤔',
  '👎',
  '😡',
  '🤝',
  '💪',
];

const PICKER_ESTIMATED_WIDTH = 220;
const PICKER_ESTIMATED_HEIGHT = 120;
const PAGE_SIZE = 4;

type Props = {
  x: number;
  y: number;
  onSelect: (emoji: string) => void;
  onClose: () => void;
};

export function EmojiPicker({ x, y, onSelect, onClose }: Props) {
  const pickerRef = useRef<HTMLDivElement>(null);

  const initialPosition = useMemo(() => {
    const chatContainer = document.querySelector('.chat-scroll-area') as HTMLElement | null;
    const padding = 10;

    let adjustedX = x;
    let adjustedY = y;

    if (chatContainer) {
      const { width, height } = chatContainer.getBoundingClientRect();

      if (adjustedX + PICKER_ESTIMATED_WIDTH > width - padding) {
        adjustedX = width - PICKER_ESTIMATED_WIDTH - padding;
      }
      if (adjustedX < padding) {
        adjustedX = padding;
      }

      if (adjustedY + PICKER_ESTIMATED_HEIGHT > height - padding) {
        adjustedY = height - PICKER_ESTIMATED_HEIGHT - padding;
      }
      if (adjustedY < padding) {
        adjustedY = padding;
      }
    }

    return { x: adjustedX, y: adjustedY };
  }, [x, y]);

  const [position, setPosition] = useState(initialPosition);
  const [startIndex, setStartIndex] = useState(0);

  const visibleEmojis = useMemo(
    () => EMOJI_LIST.slice(startIndex, startIndex + PAGE_SIZE),
    [startIndex],
  );

  const canPrev = startIndex > 0;
  const canNext = startIndex + PAGE_SIZE < EMOJI_LIST.length;

  const handlePrev = useCallback(() => {
    setStartIndex((prev) => Math.max(0, prev - PAGE_SIZE));
  }, []);

  const handleNext = useCallback(() => {
    setStartIndex((prev) => {
      const next = prev + PAGE_SIZE;
      if (next >= EMOJI_LIST.length) {
        return prev;
      }
      return next;
    });
  }, []);

  useLayoutEffect(() => {
    if (!pickerRef.current) return;

    const pickerRect = pickerRef.current.getBoundingClientRect();
    const chatContainer = document.querySelector('.chat-scroll-area') as HTMLElement | null;
    const padding = 10;

    let adjustedX = x;
    let adjustedY = y;

    if (chatContainer) {
      const { width, height } = chatContainer.getBoundingClientRect();

      if (adjustedX + pickerRect.width > width - padding) {
        adjustedX = width - pickerRect.width - padding;
      }
      if (adjustedX < padding) {
        adjustedX = padding;
      }

      if (adjustedY + pickerRect.height > height - padding) {
        adjustedY = height - pickerRect.height - padding;
      }
      if (adjustedY < padding) {
        adjustedY = padding;
      }
    }

    setPosition((prevPosition) => {
      if (prevPosition.x === adjustedX && prevPosition.y === adjustedY) {
        return prevPosition;
      }
      return { x: adjustedX, y: adjustedY };
    });
  }, [x, y]);

  useLayoutEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (pickerRef.current && !pickerRef.current.contains(e.target as Node)) {
        onClose();
      }
    };

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [onClose]);

  return (
    <div
      ref={pickerRef}
      className="z-[140] bg-black/95 border border-emerald-600/50 rounded-lg shadow-xl px-2 py-2 absolute"
      style={{
        left: `${position.x}px`,
        top: `${position.y}px`
      }}
      onClick={(e) => e.stopPropagation()}
    >
      <div className="flex items-center gap-1">
        <button
          type="button"
          onClick={handlePrev}
          disabled={!canPrev}
          className="w-8 h-9 rounded-md bg-emerald-950/40 border border-emerald-700/60 text-emerald-300 hover:bg-emerald-900/60 disabled:opacity-40 disabled:cursor-not-allowed transition-colors flex items-center justify-center text-sm"
        >
          ←
        </button>
        <div className="flex gap-1">
          {visibleEmojis.map((emoji) => (
            <button
              key={emoji}
              onClick={() => {
                onSelect(emoji);
                onClose();
              }}
              className="w-12 h-12 flex items-center justify-center text-xl hover:bg-emerald-500/20 rounded-md transition-colors"
            >
              {emoji}
            </button>
          ))}
        </div>
        <button
          type="button"
          onClick={handleNext}
          disabled={!canNext}
          className="w-8 h-9 rounded-md bg-emerald-950/40 border border-emerald-700/60 text-emerald-300 hover:bg-emerald-900/60 disabled:opacity-40 disabled:cursor-not-allowed transition-colors flex items-center justify-center text-sm"
        >
          →
        </button>
      </div>
    </div>
  );
}
