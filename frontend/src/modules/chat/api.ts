import {
  parseApiError,
  UNAUTHORIZED_MESSAGE,
  type ApiErrorResponse,
} from '../../shared/api/error-handler';

const API_BASE = '/api/chat';

export type MeResponse = {
  id: string;
  username: string;
};

export type UserSummary = {
  id: string;
  username: string;
};

export type ChatErrorResponse = ApiErrorResponse;

export {
  UNAUTHORIZED_MESSAGE,
  SESSION_EXPIRED_ERROR,
} from '../../shared/api/error-handler';

export async function fetchMe(token: string): Promise<MeResponse> {
  const res = await fetch(`${API_BASE}/me`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    if (res.status === 401) {
      throw new Error(UNAUTHORIZED_MESSAGE);
    }
    const errorMessage = await parseApiError(res);
    throw new Error(errorMessage);
  }

  const json = (await res.json()) as MeResponse;
  return json;
}

export async function searchUsers(
  query: string,
  token: string,
): Promise<UserSummary[]> {
  const params = new URLSearchParams({ username: query });
  const res = await fetch(`${API_BASE}/users?${params.toString()}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    if (res.status === 401) {
      throw new Error(UNAUTHORIZED_MESSAGE);
    }
    const errorMessage = await parseApiError(res);
    throw new Error(errorMessage);
  }

  const json = (await res.json()) as UserSummary[];
  return json;
}

export type IdentityKeyResponse = {
  public_key: string;
};

export async function getIdentityKey(
  userId: string,
  token: string,
): Promise<IdentityKeyResponse> {
  const res = await fetch(`${API_BASE}/users/${userId}/identity-key`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    if (res.status === 401) {
      throw new Error(UNAUTHORIZED_MESSAGE);
    }
    const errorMessage = await parseApiError(res);
    throw new Error(errorMessage);
  }

  const json = (await res.json()) as IdentityKeyResponse;
  return json;
}

export type FingerprintResponse = {
  fingerprint: string;
};

export async function getFingerprint(
  userId: string,
  token: string,
): Promise<FingerprintResponse> {
  const res = await fetch(`/api/identity/users/${userId}/fingerprint`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    credentials: 'include',
  });

  if (!res.ok) {
    if (res.status === 401) {
      throw new Error(UNAUTHORIZED_MESSAGE);
    }
    const errorMessage = await parseApiError(res);
    throw new Error(errorMessage);
  }

  const contentType = res.headers.get('content-type');
  if (!contentType || !contentType.includes('application/json')) {
    throw new Error('Неверный формат ответа от сервера');
  }

  const json = (await res.json()) as FingerprintResponse;
  return json;
}
