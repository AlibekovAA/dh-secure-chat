import { getFriendlyErrorMessage, parseApiError, type ApiErrorResponse } from '../../shared/api/error-handler';

const API_BASE = '/api/auth';

export type AuthToken = string;

export type AuthResponse = {
  token: AuthToken;
};

export type AuthErrorResponse = ApiErrorResponse;

export async function register(
  username: string,
  password: string,
  identityPubKey?: string,
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

  const res = await fetch(`${API_BASE}/register`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const errorMessage = await parseApiError(res);
    throw new Error(errorMessage);
  }

  const json = (await res.json()) as AuthResponse;
  return json;
}

export async function login(
  username: string,
  password: string,
): Promise<AuthResponse> {
  const res = await fetch(`${API_BASE}/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify({ username, password }),
  });

  if (!res.ok) {
    const errorMessage = await parseApiError(res);
    throw new Error(errorMessage);
  }

  const json = (await res.json()) as AuthResponse;
  return json;
}
