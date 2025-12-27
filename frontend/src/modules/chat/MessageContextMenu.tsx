import { useRef, useLayoutEffect, useState, useMemo } from 'react';

type Props = {
  x: number;
  y: number;
  isOwn: boolean;
  canEdit: boolean;
  onCopy: () => void;
  onReact: () => void;
  onReply?: () => void;
  onEdit?: () => void;
  onDeleteForMe?: () => void;
  onDeleteForAll?: () => void;
  onClose: () => void;
};

const MENU_ESTIMATED_WIDTH = 260;
const MENU_ESTIMATED_HEIGHT = 140;

export function MessageContextMenu({
  x,
  y,
  isOwn,
  canEdit,
  onCopy,
  onReact,
  onReply,
  onEdit,
  onDeleteForMe,
  onDeleteForAll,
  onClose,
}: Props) {
  const menuRef = useRef<HTMLDivElement>(null);

  const initialPosition = useMemo(() => {
    const chatContainer = document.querySelector('.chat-scroll-area') as HTMLElement | null;
    const padding = 10;

    let adjustedX = x;
    let adjustedY = y;

    if (chatContainer) {
      const { width, height } = chatContainer.getBoundingClientRect();

      if (adjustedX + MENU_ESTIMATED_WIDTH > width - padding) {
        adjustedX = width - MENU_ESTIMATED_WIDTH - padding;
      }
      if (adjustedX < padding) {
        adjustedX = padding;
      }

      if (adjustedY + MENU_ESTIMATED_HEIGHT > height - padding) {
        adjustedY = height - MENU_ESTIMATED_HEIGHT - padding;
      }
      if (adjustedY < padding) {
        adjustedY = padding;
      }
    }

    return { x: adjustedX, y: adjustedY };
  }, [x, y]);

  const [position, setPosition] = useState(() => ({ x: initialPosition.x, y: initialPosition.y }));

  useLayoutEffect(() => {
    if (!menuRef.current) {
      return;
    }

    const menuRect = menuRef.current.getBoundingClientRect();
    const chatContainer = document.querySelector('.chat-scroll-area') as HTMLElement | null;
    const padding = 10;

    let adjustedX = x;
    let adjustedY = y;

    if (chatContainer) {
      const { width, height } = chatContainer.getBoundingClientRect();

      if (adjustedX + menuRect.width > width - padding) {
        adjustedX = width - menuRect.width - padding;
      }
      if (adjustedX < padding) {
        adjustedX = padding;
      }

      if (adjustedY + menuRect.height > height - padding) {
        adjustedY = height - menuRect.height - padding;
      }
      if (adjustedY < padding) {
        adjustedY = padding;
      }
    }

    setPosition(current => {
      if (current.x === adjustedX && current.y === adjustedY) {
        return current;
      }
      return { x: adjustedX, y: adjustedY };
    });
  }, [x, y]);

  useLayoutEffect(() => {
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

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [onClose]);

  return (
    <div
      ref={menuRef}
      className="z-[130] bg-black/95 border border-emerald-600/50 rounded-xl shadow-2xl py-2 px-3 min-w-[200px] absolute"
      style={{
        left: `${position.x}px`,
        top: `${position.y}px`
      }}
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

      {canEdit && onEdit && (
        <button
          onClick={() => {
            onEdit();
            onClose();
          }}
          className="w-full text-left px-4 py-2 text-sm text-emerald-50 hover:bg-emerald-500/20 transition-colors"
        >
          <span>Редактировать</span>
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
