import { TOKEN_STORAGE_KEY } from '@/shared/constants';

export function removeToken(): void {
  localStorage.removeItem(TOKEN_STORAGE_KEY);
}
