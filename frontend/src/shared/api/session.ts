import { apiClient } from '@/shared/api/client';

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
    const [{ removeToken }, { clearAllKeys }] = await Promise.all([
      import('@/shared/storage/token'),
      import('@/shared/storage/indexeddb'),
    ]);

    removeToken();
    await clearAllKeys();

    try {
      localStorage.removeItem('userId');
    } catch {
      void 0;
    }
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
    showToast('Сессия истекла. Войдите снова', 'error');
  }

  attachTokenToClient(null);
  resetAuthState();
  await clearSessionStorageSideEffects();
}
