import { useRef, useEffect, useState } from 'react';

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
  '✨',
  '⭐',
  '🎯',
];

type Props = {
  x: number;
  y: number;
  onSelect: (emoji: string) => void;
  onClose: () => void;
};

export function EmojiPicker({ x, y, onSelect, onClose }: Props) {
  const pickerRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState({ x, y });

  useEffect(() => {
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

    if (pickerRef.current) {
      const rect = pickerRef.current.getBoundingClientRect();
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      let adjustedX = x;
      let adjustedY = y;

      if (x + rect.width > viewportWidth) {
        adjustedX = viewportWidth - rect.width - 10;
      }
      if (y + rect.height > viewportHeight) {
        adjustedY = viewportHeight - rect.height - 10;
      }

      setPosition({ x: adjustedX, y: adjustedY });
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [x, y, onClose]);

  return (
    <div
      ref={pickerRef}
      className="fixed z-50 bg-black/95 border border-emerald-500/40 rounded-lg shadow-lg p-2 grid grid-cols-5 gap-1"
      style={{ left: `${position.x}px`, top: `${position.y}px` }}
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
