import { apiClient } from '@/shared/api/client';
import { IDENTITY_KEY_STORAGE, MASTER_KEY_STORAGE } from '@/shared/constants';
import { MESSAGES } from '@/shared/messages';

type ToastFn = (
  message: string,
  type?: 'error' | 'success' | 'warning'
) => void;

type SessionContext = {
  showToast?: ToastFn;
  resetAuthState: () => void;
};

type SessionExpiredOptions = {
  silent?: boolean;
};

export async function clearSessionStorageSideEffects(): Promise<void> {
  try {
    const { removeToken } = await import('@/shared/storage/token');
    removeToken();
  } catch {
    void 0;
  }

  const cleanupTasks: Promise<unknown>[] = [];
  try {
    const { clearAllKeys } = await import('@/shared/storage/indexeddb');
    cleanupTasks.push(clearAllKeys());
  } catch {
    void 0;
  }
  try {
    const { clearVideoThumbnailCache } =
      await import('@/modules/chat/video-thumbnail');
    clearVideoThumbnailCache();
  } catch {
    void 0;
  }

  await Promise.allSettled(cleanupTasks);

  try {
    localStorage.removeItem('userId');
    localStorage.removeItem(IDENTITY_KEY_STORAGE);
    localStorage.removeItem(MASTER_KEY_STORAGE);
  } catch {
    void 0;
  }
}

export function attachTokenToClient(token: string | null): void {
  apiClient.setToken(token);
}

export async function handleSessionExpired(
  _error: unknown,
  ctx: SessionContext,
  options: SessionExpiredOptions = {}
): Promise<void> {
  const { showToast, resetAuthState } = ctx;
  const { silent = false } = options;

  if (!silent && showToast) {
    showToast(MESSAGES.apiErrors.sessionExpired, 'error');
  }

  attachTokenToClient(null);
  resetAuthState();
  await clearSessionStorageSideEffects();
}
