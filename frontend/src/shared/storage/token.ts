const TOKEN_STORAGE_KEY = 'auth_token';

export function removeToken(): void {
  localStorage.removeItem(TOKEN_STORAGE_KEY);
}
