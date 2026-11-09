import { useRef, useLayoutEffect, useEffect, useState, useMemo } from 'react';
import { createPortal } from 'react-dom';
import {
  MENU_WIDTH,
  MENU_ESTIMATED_HEIGHT,
  MENU_PADDING,
} from '@/shared/constants';
import {
  computeFloatingPosition,
  type AnchorRect,
} from '@/modules/chat/floating-position';
import { MESSAGES } from '@/shared/messages';

const useIsomorphicLayoutEffect =
  typeof window !== 'undefined' ? useLayoutEffect : useEffect;

type Props = {
  anchorRect: AnchorRect;
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

export function MessageContextMenu({
  anchorRect,
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

  const { left, right, top, bottom, width, height } = anchorRect;

  const initialPosition = useMemo(() => {
    if (typeof document === 'undefined') {
      return { x: left, y: bottom + 8 };
    }

    const chatContainer = document.querySelector(
      '.chat-scroll-area'
    ) as HTMLElement | null;
    const padding = MENU_PADDING;

    const offset = 8;
    const estWidth = MENU_WIDTH;
    const estHeight = MENU_ESTIMATED_HEIGHT;

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

  const [position, setPosition] = useState(() => ({
    x: initialPosition.x,
    y: initialPosition.y,
  }));

  useIsomorphicLayoutEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }
    if (!menuRef.current) {
      return;
    }

    const menuRect = menuRef.current.getBoundingClientRect();
    const chatContainer = document.querySelector(
      '.chat-scroll-area'
    ) as HTMLElement | null;
    const padding = MENU_PADDING;

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
      popupWidth: menuRect.width,
      popupHeight: menuRect.height,
      padding,
      offset,
      boundsRect,
    });

    setPosition((current) => {
      if (current.x === adjustedX && current.y === adjustedY) {
        return current;
      }
      return { x: adjustedX, y: adjustedY };
    });
  }, [left, right, top, bottom, width, height, isOwn]);

  useIsomorphicLayoutEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }
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

  useEffect(() => {
    const menu = menuRef.current;
    if (!menu) return;
    const items = Array.from(
      menu.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')
    );
    items[0]?.focus();

    const handleKeyDown = (event: KeyboardEvent) => {
      const currentIndex = items.indexOf(
        document.activeElement as HTMLButtonElement
      );
      let nextIndex: number | null = null;
      if (event.key === 'ArrowDown')
        nextIndex = (currentIndex + 1) % items.length;
      if (event.key === 'ArrowUp')
        nextIndex = (currentIndex - 1 + items.length) % items.length;
      if (event.key === 'Home') nextIndex = 0;
      if (event.key === 'End') nextIndex = items.length - 1;
      if (nextIndex === null) return;
      event.preventDefault();
      items[nextIndex]?.focus();
    };

    menu.addEventListener('keydown', handleKeyDown);
    return () => menu.removeEventListener('keydown', handleKeyDown);
  }, []);

  if (typeof document === 'undefined') {
    return null;
  }

  return createPortal(
    <>
      <div
        className="fixed inset-0 bg-black/40 backdrop-blur-[0.5px] z-[120] animate-[fadeIn_0.15s_ease-out]"
        onClick={onClose}
      />
      <div
        ref={menuRef}
        className="z-[130] bg-[var(--color-surface-raised)] border border-slate-700 rounded-lg shadow-xl shadow-black/40 py-2 px-2 min-w-[200px] fixed"
        style={{
          left: `${position.x}px`,
          top: `${position.y}px`,
          width: `${MENU_WIDTH}px`,
        }}
        onClick={(e) => e.stopPropagation()}
        role="menu"
        aria-label={MESSAGES.chat.contextMenu.aria.label}
      >
        <button
          type="button"
          role="menuitem"
          onClick={() => {
            onCopy();
            onClose();
          }}
          className="w-full text-left px-4 py-2 text-sm text-emerald-50 hover:bg-emerald-500/20 transition-colors"
        >
          <span>{MESSAGES.chat.contextMenu.actions.copy}</span>
        </button>

        <button
          type="button"
          role="menuitem"
          onClick={() => {
            onReact();
            onClose();
          }}
          className="w-full text-left px-4 py-2 text-sm text-emerald-50 hover:bg-emerald-500/20 transition-colors"
        >
          <span>{MESSAGES.chat.contextMenu.actions.react}</span>
        </button>

        {onReply && (
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onReply();
              onClose();
            }}
            className="w-full text-left px-4 py-2 text-sm text-emerald-50 hover:bg-emerald-500/20 transition-colors"
          >
            <span>{MESSAGES.chat.contextMenu.actions.reply}</span>
          </button>
        )}

        {canEdit && onEdit && (
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onEdit();
              onClose();
            }}
            className="w-full text-left px-4 py-2 text-sm text-emerald-50 hover:bg-emerald-500/20 transition-colors"
          >
            <span>{MESSAGES.chat.contextMenu.actions.edit}</span>
          </button>
        )}

        {isOwn && onDeleteForMe && (
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onDeleteForMe();
              onClose();
            }}
            className="w-full text-left px-4 py-2 text-sm text-red-400 hover:bg-red-500/20 transition-colors"
          >
            <span>{MESSAGES.chat.contextMenu.actions.deleteSelf}</span>
          </button>
        )}

        {isOwn && onDeleteForAll && (
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onDeleteForAll();
              onClose();
            }}
            className="w-full text-left px-4 py-2 text-sm text-red-400 hover:bg-red-500/20 transition-colors"
          >
            <span>{MESSAGES.chat.contextMenu.actions.deleteEveryone}</span>
          </button>
        )}
      </div>
    </>,
    document.body
  );
}
