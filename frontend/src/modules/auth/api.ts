import { apiClient } from '@/shared/api/client';
import type { ApiErrorResponse } from '@/shared/api/error-handler';

const API_BASE = '/api/auth';

export type AuthToken = string;

export type AuthResponse = {
  token: AuthToken;
};

export type AuthErrorResponse = ApiErrorResponse;

export async function register(
  username: string,
  password: string,
  identityPubKey?: string
): Promise<AuthResponse> {
  const body: {
    username: string;
    password: string;
    identity_pub_key?: string;
  } = {
    username,
    password,
  };
  if (identityPubKey) {
    body.identity_pub_key = identityPubKey;
  }

  return apiClient.post<AuthResponse>(`${API_BASE}/register`, body, {
    credentials: 'include',
  });
}

export async function login(
  username: string,
  password: string
): Promise<AuthResponse> {
  return apiClient.post<AuthResponse>(
    `${API_BASE}/login`,
    { username, password },
    { credentials: 'include' }
  );
}
