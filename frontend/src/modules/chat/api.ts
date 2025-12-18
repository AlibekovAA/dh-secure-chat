const API_BASE = '/api/chat';

export type MeResponse = {
  id: string;
  username: string;
};

export type UserSummary = {
  id: string;
  username: string;
};

export const UNAUTHORIZED_MESSAGE = 'unauthorized';
export const SESSION_EXPIRED_ERROR = 'session_expired';

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
    throw new Error('Failed to load profile');
  }

  return (await res.json()) as MeResponse;
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
    throw new Error('Search failed');
  }

  return (await res.json()) as UserSummary[];
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
    throw new Error('Failed to get identity key');
  }

  return (await res.json()) as IdentityKeyResponse;
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
    const contentType = res.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      const error = await res.json().catch(() => ({}));
      throw new Error(
        typeof error === 'object' && 'error' in error
          ? String(error.error)
          : 'Failed to get fingerprint',
      );
    }
    throw new Error(
      `Failed to get fingerprint: ${res.status} ${res.statusText}`,
    );
  }

  const contentType = res.headers.get('content-type');
  if (!contentType || !contentType.includes('application/json')) {
    throw new Error('Invalid response format from server');
  }

  return (await res.json()) as FingerprintResponse;
}
