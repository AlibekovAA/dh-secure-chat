import { useRef, useLayoutEffect, useState, useMemo } from 'react';

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
  '😊',
  '😍',
  '🤔',
  '👎',
  '😡',
  '🤝',
  '💪',
];

const PICKER_ESTIMATED_WIDTH = 240;
const PICKER_ESTIMATED_HEIGHT = 220;

type Props = {
  x: number;
  y: number;
  onSelect: (emoji: string) => void;
  onClose: () => void;
};

export function EmojiPicker({ x, y, onSelect, onClose }: Props) {
  const pickerRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLElement | null>(null);

  const initialPosition = useMemo(() => {
    const chatContainer = document.querySelector('[class*="overflow-y-auto"][class*="relative"]') as HTMLElement;
    containerRef.current = chatContainer;

    if (chatContainer) {
      const containerRect = chatContainer.getBoundingClientRect();
      const padding = 8;

      let adjustedX = x;
      let adjustedY = y;

      if (adjustedX + PICKER_ESTIMATED_WIDTH > containerRect.width - padding) {
        adjustedX = containerRect.width - PICKER_ESTIMATED_WIDTH - padding;
      }
      if (adjustedX < padding) {
        adjustedX = padding;
      }

      if (adjustedY + PICKER_ESTIMATED_HEIGHT > containerRect.height - padding) {
        adjustedY = containerRect.height - PICKER_ESTIMATED_HEIGHT - padding;
      }
      if (adjustedY < padding) {
        adjustedY = padding;
      }

      return { x: adjustedX, y: adjustedY, isRelative: true };
    } else {
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;
      const padding = 8;

      let adjustedX = x;
      let adjustedY = y;

      if (x + PICKER_ESTIMATED_WIDTH > viewportWidth - padding) {
        adjustedX = viewportWidth - PICKER_ESTIMATED_WIDTH - padding;
      }
      if (x < padding) {
        adjustedX = padding;
      }

      if (y + PICKER_ESTIMATED_HEIGHT > viewportHeight - padding) {
        adjustedY = viewportHeight - PICKER_ESTIMATED_HEIGHT - padding;
      }
      if (y < padding) {
        adjustedY = padding;
      }

      return { x: adjustedX, y: adjustedY, isRelative: false };
    }
  }, [x, y]);

  const [position, setPosition] = useState(initialPosition);
  const [isRelativeToContainer, setIsRelativeToContainer] = useState(initialPosition.isRelative);

  useLayoutEffect(() => {
    if (!pickerRef.current) return;

    const pickerRect = pickerRef.current.getBoundingClientRect();
    const chatContainer = containerRef.current || pickerRef.current.closest('[class*="overflow-y-auto"]') as HTMLElement;

    let adjustedX: number;
    let adjustedY: number;
    let shouldBeRelative: boolean;

    if (chatContainer) {
      const containerRect = chatContainer.getBoundingClientRect();
      const padding = 8;

      adjustedX = x;
      adjustedY = y;

      if (adjustedX + pickerRect.width > containerRect.width - padding) {
        adjustedX = containerRect.width - pickerRect.width - padding;
      }
      if (adjustedX < padding) {
        adjustedX = padding;
      }

      if (adjustedY + pickerRect.height > containerRect.height - padding) {
        adjustedY = containerRect.height - pickerRect.height - padding;
      }
      if (adjustedY < padding) {
        adjustedY = padding;
      }

      shouldBeRelative = true;
    } else {
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;
      const padding = 8;

      adjustedX = x;
      adjustedY = y;

      if (x + pickerRect.width > viewportWidth - padding) {
        adjustedX = viewportWidth - pickerRect.width - padding;
      }
      if (x < padding) {
        adjustedX = padding;
      }

      if (y + pickerRect.height > viewportHeight - padding) {
        adjustedY = viewportHeight - pickerRect.height - padding;
      }
      if (y < padding) {
        adjustedY = padding;
      }

      shouldBeRelative = false;
    }

    setPosition((prevPosition) => {
      if (prevPosition.x === adjustedX && prevPosition.y === adjustedY && prevPosition.isRelative === shouldBeRelative) {
        return prevPosition;
      }
      return { x: adjustedX, y: adjustedY, isRelative: shouldBeRelative };
    });

    setIsRelativeToContainer((prev) => {
      if (prev !== shouldBeRelative) {
        return shouldBeRelative;
      }
      return prev;
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
      className={`z-50 bg-black/95 border border-emerald-500/40 rounded-lg shadow-lg p-2 grid grid-cols-5 gap-1 ${isRelativeToContainer ? 'absolute' : 'fixed'}`}
      style={{
        left: `${position.x}px`,
        top: `${position.y}px`
      }}
      onClick={(e) => e.stopPropagation()}
    >
      {EMOJI_LIST.map((emoji) => (
        <button
          key={emoji}
          onClick={() => {
            onSelect(emoji);
            onClose();
          }}
          className="w-10 h-10 flex items-center justify-center text-xl hover:bg-emerald-500/20 rounded transition-colors"
        >
          {emoji}
        </button>
      ))}
    </div>
  );
}
