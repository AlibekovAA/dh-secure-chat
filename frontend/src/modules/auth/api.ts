const API_BASE = '/api/auth';

export type AuthToken = string;

export type AuthResponse = {
  token: AuthToken;
};

export type AuthError = {
  error: string;
};

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

  const json = (await res.json()) as AuthResponse | AuthError;

  if (!res.ok) {
    throw new Error('error' in json ? json.error : 'Registration failed');
  }

  return json as AuthResponse;
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

  const json = (await res.json()) as AuthResponse | AuthError;

  if (!res.ok) {
    throw new Error('error' in json ? json.error : 'Login failed');
  }

  return json as AuthResponse;
}
