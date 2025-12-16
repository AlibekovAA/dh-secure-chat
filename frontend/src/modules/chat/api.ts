const API_BASE = '/api/chat';

export type MeResponse = {
  id: string;
  username: string;
};

export type UserSummary = {
  id: string;
  username: string;
};

export async function fetchMe(token: string): Promise<MeResponse> {
  const res = await fetch(`${API_BASE}/me`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
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
    throw new Error('Failed to get identity key');
  }

  return (await res.json()) as IdentityKeyResponse;
}
