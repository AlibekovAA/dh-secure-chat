import {
  useRef,
  useLayoutEffect,
  useEffect,
  useState,
  useMemo,
  useCallback,
} from 'react';
import { createPortal } from 'react-dom';
import {
  EMOJI_LIST,
  EMOJI_PICKER_ESTIMATED_HEIGHT,
  EMOJI_PICKER_WIDTH,
  EMOJI_PICKER_PADDING,
  EMOJI_PICKER_PAGE_SIZE,
} from '@/shared/constants';
import {
  computeFloatingPosition,
  type AnchorRect,
} from '@/modules/chat/floating-position';

const useIsomorphicLayoutEffect =
  typeof window !== 'undefined' ? useLayoutEffect : useEffect;

type Props = {
  anchorRect: AnchorRect;
  isOwn: boolean;
  onSelect: (emoji: string) => void;
  onClose: () => void;
};

export function EmojiPicker({ anchorRect, isOwn, onSelect, onClose }: Props) {
  const pickerRef = useRef<HTMLDivElement>(null);

  const { left, right, top, bottom, width, height } = anchorRect;

  const initialPosition = useMemo(() => {
    if (typeof document === 'undefined') {
      return { x: anchorRect.left, y: anchorRect.bottom + 8 };
    }

    const chatContainer = document.querySelector(
      '.chat-scroll-area'
    ) as HTMLElement | null;
    const padding = EMOJI_PICKER_PADDING;

    const offset = 8;
    const estWidth = EMOJI_PICKER_WIDTH;
    const estHeight = EMOJI_PICKER_ESTIMATED_HEIGHT;

    const boundsRect = chatContainer
      ? chatContainer.getBoundingClientRect()
      : {
          left: 0,
          top: 0,
          right: window.innerWidth,
          bottom: window.innerHeight,
        };

    return computeFloatingPosition({
      anchorRect: { left, right, top, bottom, width, height },
      isOwn,
      popupWidth: estWidth,
      popupHeight: estHeight,
      padding,
      offset,
      boundsRect,
    });
  }, [left, right, top, bottom, width, height, isOwn]);

  const [position, setPosition] = useState(initialPosition);
  const [startIndex, setStartIndex] = useState(0);

  const visibleEmojis = useMemo(
    () => EMOJI_LIST.slice(startIndex, startIndex + EMOJI_PICKER_PAGE_SIZE),
    [startIndex]
  );

  const canPrev = startIndex > 0;
  const canNext = startIndex + EMOJI_PICKER_PAGE_SIZE < EMOJI_LIST.length;

  const handlePrev = useCallback(() => {
    setStartIndex((prev) => Math.max(0, prev - EMOJI_PICKER_PAGE_SIZE));
  }, []);

  const handleNext = useCallback(() => {
    setStartIndex((prev) => {
      const next = prev + EMOJI_PICKER_PAGE_SIZE;
      if (next >= EMOJI_LIST.length) {
        return prev;
      }
      return next;
    });
  }, []);

  useIsomorphicLayoutEffect(() => {
    if (typeof document === 'undefined') return;
    if (!pickerRef.current) return;

    const pickerRect = pickerRef.current.getBoundingClientRect();
    const chatContainer = document.querySelector(
      '.chat-scroll-area'
    ) as HTMLElement | null;
    const padding = EMOJI_PICKER_PADDING;

    const offset = 8;
    const boundsRect = chatContainer
      ? chatContainer.getBoundingClientRect()
      : {
          left: 0,
          top: 0,
          right: window.innerWidth,
          bottom: window.innerHeight,
        };

    const { x: adjustedX, y: adjustedY } = computeFloatingPosition({
      anchorRect: { left, right, top, bottom, width, height },
      isOwn,
      popupWidth: pickerRect.width,
      popupHeight: pickerRect.height,
      padding,
      offset,
      boundsRect,
    });

    setPosition((prevPosition) => {
      if (prevPosition.x === adjustedX && prevPosition.y === adjustedY) {
        return prevPosition;
      }
      return { x: adjustedX, y: adjustedY };
    });
  }, [left, right, top, bottom, width, height, isOwn]);

  useIsomorphicLayoutEffect(() => {
    if (typeof document === 'undefined') return;
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

  if (typeof document === 'undefined') {
    return null;
  }

  return createPortal(
    <div
      ref={pickerRef}
      className="z-[140] bg-black/95 border border-emerald-600/50 rounded-lg shadow-xl px-2 py-2 fixed"
      style={{
        left: `${position.x}px`,
        top: `${position.y}px`,
        width: `${EMOJI_PICKER_WIDTH}px`,
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
    </div>,
    document.body
  );
}
