import { TOKEN_STORAGE_KEY } from '../constants';

export function removeToken(): void {
  localStorage.removeItem(TOKEN_STORAGE_KEY);
}
