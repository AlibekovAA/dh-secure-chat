import { parseApiError } from '@/shared/api/error-handler';
import {
  SESSION_EXPIRED_ERROR,
  UNAUTHORIZED_MESSAGE,
} from '@/shared/constants';

type RefreshTokenCallback = () => Promise<string | null>;
type OnTokenExpiredCallback = () => void;

class ApiClient {
  private token: string | null = null;
  private refreshTokenFn: RefreshTokenCallback | null = null;
  private onTokenExpired: OnTokenExpiredCallback | null = null;

  setToken(token: string | null): void {
    this.token = token;
  }

  setRefreshTokenFn(fn: RefreshTokenCallback): void {
    this.refreshTokenFn = fn;
  }

  setOnTokenExpired(fn: OnTokenExpiredCallback): void {
    this.onTokenExpired = fn;
  }

  private async refreshToken(): Promise<string | null> {
    if (!this.refreshTokenFn) {
      return null;
    }
    const newToken = await this.refreshTokenFn();
    if (newToken) {
      this.token = newToken;
    }
    return newToken;
  }

  private async request<T>(
    url: string,
    options: RequestInit = {},
    retry = true
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    };

    if (this.token && !headers['Authorization']) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers,
      credentials: 'include',
    });

    if (!response.ok) {
      const isRefreshEndpoint = url.includes('/api/auth/refresh');

      if (response.status === 401 && isRefreshEndpoint) {
        this.token = null;
        if (this.onTokenExpired) {
          this.onTokenExpired();
        }
        const error = new Error(SESSION_EXPIRED_ERROR);
        (error as { silent?: boolean }).silent = true;
        throw error;
      }

      const isAuthEndpoint =
        url.includes('/api/auth/login') || url.includes('/api/auth/register');

      if (
        response.status === 401 &&
        retry &&
        this.refreshTokenFn &&
        this.token &&
        !isRefreshEndpoint &&
        !isAuthEndpoint
      ) {
        const newToken = await this.refreshToken();
        if (newToken) {
          headers['Authorization'] = `Bearer ${newToken}`;
          const retryResponse = await fetch(url, {
            ...options,
            headers,
            credentials: 'include',
          });

          if (!retryResponse.ok) {
            if (retryResponse.status === 401) {
              this.token = null;
              if (this.onTokenExpired) {
                this.onTokenExpired();
              }
              throw new Error(UNAUTHORIZED_MESSAGE);
            }
            const errorMessage = await parseApiError(retryResponse);
            throw new Error(errorMessage.message);
          }

          const json = (await retryResponse.json()) as T;
          return json;
        } else {
          this.token = null;
          if (this.onTokenExpired) {
            this.onTokenExpired();
          }
          throw new Error(SESSION_EXPIRED_ERROR);
        }
      }

      if (response.status === 401) {
        if (!isAuthEndpoint) {
          this.token = null;
          if (this.onTokenExpired) {
            this.onTokenExpired();
          }
        }

        const errorMessage = await parseApiError(response);
        throw new Error(errorMessage.message);
      }

      const errorMessage = await parseApiError(response);
      throw new Error(errorMessage.message);
    }

    const json = (await response.json()) as T;
    return json;
  }

  async get<T>(url: string, options?: RequestInit): Promise<T> {
    return this.request<T>(url, { ...options, method: 'GET' });
  }

  async post<T>(
    url: string,
    body?: unknown,
    options?: RequestInit
  ): Promise<T> {
    return this.request<T>(url, {
      ...options,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  async put<T>(url: string, body?: unknown, options?: RequestInit): Promise<T> {
    return this.request<T>(url, {
      ...options,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  async delete<T>(url: string, options?: RequestInit): Promise<T> {
    return this.request<T>(url, { ...options, method: 'DELETE' });
  }
}

export const apiClient = new ApiClient();
