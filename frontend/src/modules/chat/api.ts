import { apiClient } from '../../shared/api/client';
import type { ApiErrorResponse } from '../../shared/api/error-handler';

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

export async function fetchMe(): Promise<MeResponse> {
  return apiClient.get<MeResponse>(`${API_BASE}/me`);
}

export async function searchUsers(query: string): Promise<UserSummary[]> {
  const params = new URLSearchParams({ username: query });
  return apiClient.get<UserSummary[]>(`${API_BASE}/users?${params.toString()}`);
}

export type IdentityKeyResponse = {
  public_key: string;
};

export async function getIdentityKey(
  userId: string,
): Promise<IdentityKeyResponse> {
  return apiClient.get<IdentityKeyResponse>(
    `${API_BASE}/users/${userId}/identity-key`,
  );
}

export type FingerprintResponse = {
  fingerprint: string;
};

export async function getFingerprint(
  userId: string,
): Promise<FingerprintResponse> {
  return apiClient.get<FingerprintResponse>(
    `/api/identity/users/${userId}/fingerprint`,
  );
}
