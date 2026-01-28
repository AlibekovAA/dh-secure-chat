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
      return { x: anchorRect.left, y: anchorRect.bottom + 8 };
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
        className="z-[130] bg-black/95 border border-emerald-600/50 rounded-xl shadow-2xl py-2 px-3 min-w-[200px] fixed"
        style={{
          left: `${position.x}px`,
          top: `${position.y}px`,
          width: `${MENU_WIDTH}px`,
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
          <span>{MESSAGES.chat.contextMenu.actions.copy}</span>
        </button>

        <button
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
