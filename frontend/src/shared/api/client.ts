import { parseApiError } from '@/shared/api/error-handler';
import { SESSION_EXPIRED_ERROR } from '@/shared/constants';

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
        const error = new Error(SESSION_EXPIRED_ERROR) as Error & {
          code?: string;
          silent?: boolean;
        };
        error.code = 'REFRESH_TOKEN_EXPIRED';
        error.silent = true;
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
              const err = new Error(SESSION_EXPIRED_ERROR) as Error & {
                code?: string;
              };
              err.code = 'UNAUTHORIZED';
              throw err;
            }
            const appErr = await parseApiError(retryResponse);
            const errRetry = new Error(appErr.message) as Error & {
              code?: string;
              statusCode?: number;
            };
            errRetry.code = appErr.code;
            errRetry.statusCode = appErr.statusCode;
            throw errRetry;
          }

          const json = (await retryResponse.json()) as T;
          return json;
        } else {
          this.token = null;
          if (this.onTokenExpired) {
            this.onTokenExpired();
          }
          const errExpired = new Error(SESSION_EXPIRED_ERROR) as Error & {
            code?: string;
          };
          errExpired.code = 'REFRESH_TOKEN_EXPIRED';
          throw errExpired;
        }
      }

      if (response.status === 401) {
        if (!isAuthEndpoint) {
          this.token = null;
          if (this.onTokenExpired) {
            this.onTokenExpired();
          }
        }

        const appErr = await parseApiError(response);
        const err401 = new Error(appErr.message) as Error & {
          code?: string;
          statusCode?: number;
        };
        err401.code = appErr.code;
        err401.statusCode = appErr.statusCode;
        throw err401;
      }

      const appErr = await parseApiError(response);
      const err = new Error(appErr.message) as Error & {
        code?: string;
        statusCode?: number;
      };
      err.code = appErr.code;
      err.statusCode = appErr.statusCode;
      throw err;
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
