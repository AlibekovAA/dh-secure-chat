import { useRef, useLayoutEffect, useState, useMemo } from 'react';

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

const MENU_ESTIMATED_WIDTH = 180;
const MENU_ESTIMATED_HEIGHT = 200;

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
  const containerRef = useRef<HTMLElement | null>(null);

  const initialPosition = useMemo(() => {
    const chatContainer = document.querySelector('[class*="overflow-y-auto"][class*="relative"]') as HTMLElement;
    containerRef.current = chatContainer;

    if (chatContainer) {
      const containerRect = chatContainer.getBoundingClientRect();
      const padding = 8;

      let adjustedX = x;
      let adjustedY = y;

      if (adjustedX + MENU_ESTIMATED_WIDTH > containerRect.width - padding) {
        adjustedX = containerRect.width - MENU_ESTIMATED_WIDTH - padding;
      }
      if (adjustedX < padding) {
        adjustedX = padding;
      }

      if (adjustedY + MENU_ESTIMATED_HEIGHT > containerRect.height - padding) {
        adjustedY = containerRect.height - MENU_ESTIMATED_HEIGHT - padding;
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

      if (x + MENU_ESTIMATED_WIDTH > viewportWidth - padding) {
        adjustedX = viewportWidth - MENU_ESTIMATED_WIDTH - padding;
      }
      if (x < padding) {
        adjustedX = padding;
      }

      if (y + MENU_ESTIMATED_HEIGHT > viewportHeight - padding) {
        adjustedY = viewportHeight - MENU_ESTIMATED_HEIGHT - padding;
      }
      if (y < padding) {
        adjustedY = padding;
      }

      return { x: adjustedX, y: adjustedY, isRelative: false };
    }
  }, [x, y]);

  const [position, setPosition] = useState(() => ({ x: initialPosition.x, y: initialPosition.y }));
  const [isRelativeToContainer, setIsRelativeToContainer] = useState(initialPosition.isRelative);

  useLayoutEffect(() => {
    if (menuRef.current) {
      const menuRect = menuRef.current.getBoundingClientRect();
      const chatContainer = containerRef.current || menuRef.current.closest('[class*="overflow-y-auto"]') as HTMLElement;

      if (chatContainer) {
        const containerRect = chatContainer.getBoundingClientRect();
        const padding = 8;

        let adjustedX = x;
        let adjustedY = y;

        if (adjustedX + menuRect.width > containerRect.width - padding) {
          adjustedX = containerRect.width - menuRect.width - padding;
        }
        if (adjustedX < padding) {
          adjustedX = padding;
        }

        if (adjustedY + menuRect.height > containerRect.height - padding) {
          adjustedY = containerRect.height - menuRect.height - padding;
        }
        if (adjustedY < padding) {
          adjustedY = padding;
        }

        if (adjustedX !== position.x || adjustedY !== position.y) {
          setPosition({ x: adjustedX, y: adjustedY });
        }
        if (!isRelativeToContainer) {
          setIsRelativeToContainer(true);
        }
      } else {
        const viewportWidth = window.innerWidth;
        const viewportHeight = window.innerHeight;
        const padding = 8;

        let adjustedX = x;
        let adjustedY = y;

        if (x + menuRect.width > viewportWidth - padding) {
          adjustedX = viewportWidth - menuRect.width - padding;
        }
        if (x < padding) {
          adjustedX = padding;
        }

        if (y + menuRect.height > viewportHeight - padding) {
          adjustedY = viewportHeight - menuRect.height - padding;
        }
        if (y < padding) {
          adjustedY = padding;
        }

        if (adjustedX !== position.x || adjustedY !== position.y) {
          setPosition({ x: adjustedX, y: adjustedY });
        }
        if (isRelativeToContainer) {
          setIsRelativeToContainer(false);
        }
      }
    }
  }, [x, y, position.x, position.y, isRelativeToContainer]);

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
      className={`z-50 bg-black/95 border border-emerald-500/40 rounded-lg shadow-lg py-1 min-w-[160px] ${isRelativeToContainer ? 'absolute' : 'fixed'}`}
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
