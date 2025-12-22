import { useRef, useEffect, useState } from 'react';

type Props = {
  x: number;
  y: number;
  isOwn: boolean;
  canEdit: boolean;
  onCopy: () => void;
  onReact: () => void;
  onReply?: () => void;
  onDeleteForMe?: () => void;
  onDeleteForAll?: () => void;
  onClose: () => void;
};

export function MessageContextMenu({
  x,
  y,
  isOwn,
  canEdit,
  onCopy,
  onReact,
  onReply,
  onDeleteForMe,
  onDeleteForAll,
  onClose,
}: Props) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState({ x, y });

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
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

    if (menuRef.current) {
      const rect = menuRef.current.getBoundingClientRect();
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
      ref={menuRef}
      className="fixed z-50 bg-black/95 border border-emerald-500/40 rounded-lg shadow-lg py-1 min-w-[160px]"
      style={{ left: `${position.x}px`, top: `${position.y}px` }}
      onClick={(e) => e.stopPropagation()}
    >
      <button
        onClick={() => {
          onCopy();
          onClose();
        }}
        className="w-full text-left px-4 py-2 text-sm text-emerald-50 hover:bg-emerald-500/20 transition-colors"
      >
        <span>Копировать</span>
      </button>

      <button
        onClick={() => {
          onReact();
          onClose();
        }}
        className="w-full text-left px-4 py-2 text-sm text-emerald-50 hover:bg-emerald-500/20 transition-colors"
      >
        <span>Отреагировать</span>
      </button>

      {onReply && (
        <button
          onClick={() => {
            onReply();
            onClose();
          }}
          className="w-full text-left px-4 py-2 text-sm text-emerald-50 hover:bg-emerald-500/20 transition-colors"
        >
          <span>Ответить</span>
        </button>
      )}

      {isOwn && onDeleteForMe && (
        <button
          onClick={() => {
          onDeleteForMe();
            onClose();
          }}
          className="w-full text-left px-4 py-2 text-sm text-red-400 hover:bg-red-500/20 transition-colors"
        >
        <span>Удалить только у себя</span>
      </button>
      )}

      {isOwn && onDeleteForAll && (
      <button
        onClick={() => {
          onDeleteForAll();
          onClose();
        }}
        className="w-full text-left px-4 py-2 text-sm text-red-400 hover:bg-red-500/20 transition-colors"
      >
        <span>Удалить у всех</span>
        </button>
      )}
    </div>
  );
}
