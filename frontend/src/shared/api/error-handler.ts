import { HTTP_SERVER_ERROR_THRESHOLD } from '@/shared/constants';
import { MESSAGES } from '@/shared/messages';

export type ApiErrorEnvelope = {
  code: string;
  message: string;
  details?: Record<string, unknown>;
  trace_id?: string;
};

export type ApiErrorResponse = ApiErrorEnvelope & {
  error?: string;
};

export type AppError = {
  message: string;
  code?: string;
  originalError?: unknown;
  statusCode?: number;
  isRetryable?: boolean;
};

export const ErrorCode = {
  UNAUTHORIZED: 'UNAUTHORIZED',
  SESSION_EXPIRED: 'SESSION_EXPIRED',
  INVALID_CREDENTIALS: 'INVALID_CREDENTIALS',
  USERNAME_TAKEN: 'USERNAME_TAKEN',
  VALIDATION_ERROR: 'VALIDATION_ERROR',
  VALIDATION_USERNAME_LENGTH: 'VALIDATION_USERNAME_LENGTH',
  VALIDATION_PASSWORD_LENGTH: 'VALIDATION_PASSWORD_LENGTH',
  VALIDATION_USERNAME_CHARS: 'VALIDATION_USERNAME_CHARS',
  VALIDATION_PASSWORD_LATIN_DIGIT: 'VALIDATION_PASSWORD_LATIN_DIGIT',
  NETWORK_ERROR: 'NETWORK_ERROR',
  SERVICE_UNAVAILABLE: 'SERVICE_UNAVAILABLE',
  FILE_TOO_LARGE: 'FILE_TOO_LARGE',
  USER_NOT_FOUND: 'USER_NOT_FOUND',
  IDENTITY_KEY_NOT_FOUND: 'IDENTITY_KEY_NOT_FOUND',
  UNKNOWN_ERROR: 'UNKNOWN_ERROR',
} as const;

export type ApiErrorCode = keyof typeof ErrorCode | string;

const CODE_TO_USER_MESSAGE: Record<string, string> = {
  UNKNOWN: MESSAGES.apiErrors.generalError,
  INVALID_CREDENTIALS: MESSAGES.apiErrors.invalidCredentials,
  USERNAME_TAKEN: MESSAGES.apiErrors.usernameTaken,
  USERNAME_ALREADY_EXISTS: MESSAGES.apiErrors.usernameTaken,
  VALIDATION_FAILED: MESSAGES.apiErrors.badRequest,
  VALIDATION_USERNAME_LENGTH: MESSAGES.apiErrors.usernameBetween,
  VALIDATION_PASSWORD_LENGTH: MESSAGES.apiErrors.passwordBetween,
  VALIDATION_USERNAME_CHARS: MESSAGES.apiErrors.usernameAllowedChars,
  VALIDATION_PASSWORD_LATIN_DIGIT:
    MESSAGES.apiErrors.passwordMustContainLatinAndDigit,
  EMPTY_QUERY: MESSAGES.apiErrors.queryEmpty,
  EMPTY_UUID: MESSAGES.apiErrors.badRequest,
  QUERY_TOO_LONG: MESSAGES.apiErrors.queryTooLong,
  INVALID_REFRESH_TOKEN: MESSAGES.apiErrors.sessionExpired,
  REFRESH_TOKEN_EXPIRED: MESSAGES.apiErrors.sessionExpired,
  MISSING_REFRESH_TOKEN: MESSAGES.apiErrors.sessionExpired,
  MISSING_AUTHORIZATION: MESSAGES.apiErrors.sessionExpired,
  INVALID_TOKEN: MESSAGES.apiErrors.sessionExpired,
  TOKEN_REVOKED: MESSAGES.apiErrors.sessionExpired,
  UNAUTHORIZED: MESSAGES.apiErrors.sessionExpired,
  SESSION_EXPIRED: MESSAGES.apiErrors.sessionExpired,
  INVALID_TOKEN_SIGNING_METHOD: MESSAGES.apiErrors.sessionExpired,
  INVALID_TOKEN_CLAIMS: MESSAGES.apiErrors.sessionExpired,
  MISSING_TOKEN_CLAIMS: MESSAGES.apiErrors.sessionExpired,
  SERVICE_UNAVAILABLE: MESSAGES.apiErrors.serviceUnavailable,
  CIRCUIT_OPEN: MESSAGES.apiErrors.serviceUnavailable,
  FILE_SIZE_EXCEEDED: MESSAGES.apiErrors.fileTooLarge,
  INVALID_FILE_SIZE: MESSAGES.apiErrors.fileTooLarge,
  INVALID_TOTAL_CHUNKS: MESSAGES.apiErrors.fileTooLarge,
  INVALID_MIME_TYPE: MESSAGES.apiErrors.fileTooLarge,
  MIME_TYPE_NOT_ALLOWED: MESSAGES.apiErrors.fileTooLarge,
  USER_NOT_FOUND: MESSAGES.apiErrors.userNotFound,
  IDENTITY_KEY_NOT_FOUND: MESSAGES.apiErrors.identityKeyNotFound,
  USER_GET_FAILED: MESSAGES.apiErrors.failedToLoadProfile,
  USER_SEARCH_FAILED: MESSAGES.apiErrors.searchFailed,
  IDENTITY_KEY_GET_FAILED: MESSAGES.apiErrors.failedToGetIdentityKey,
  IDENTITY_KEY_CREATE_FAILED: MESSAGES.apiErrors.failedToGetIdentityKey,
  FINGERPRINT_GET_FAILED: MESSAGES.apiErrors.failedToGetFingerprint,
  USER_NOT_CONNECTED: MESSAGES.apiErrors.connectionError,
  NETWORK_ERROR: MESSAGES.apiErrors.noServerConnection,
  SEND_TIMEOUT: MESSAGES.apiErrors.timeout,
  CLIENT_TOO_SLOW: MESSAGES.apiErrors.timeout,
  INVALID_JSON: MESSAGES.apiErrors.badRequest,
  INVALID_IDENTITY_PUB_KEY_ENCODING: MESSAGES.apiErrors.badRequest,
  INVALID_PUBLIC_KEY: MESSAGES.apiErrors.badRequest,
  INVALID_PATH: MESSAGES.apiErrors.badRequest,
  USER_ID_REQUIRED: MESSAGES.apiErrors.badRequest,
  INVALID_USER_ID_FORMAT: MESSAGES.apiErrors.badRequest,
  INVALID_PAYLOAD: MESSAGES.apiErrors.badRequest,
  UNKNOWN_MESSAGE_TYPE: MESSAGES.apiErrors.badRequest,
  TOKEN_MISSING_JTI: MESSAGES.apiErrors.badRequest,
  METHOD_NOT_ALLOWED: MESSAGES.apiErrors.badRequest,
  BAD_REQUEST: MESSAGES.apiErrors.badRequest,
  FORBIDDEN: MESSAGES.apiErrors.forbidden,
  NOT_FOUND: MESSAGES.apiErrors.notFound,
  TRANSFER_NOT_FOUND: MESSAGES.apiErrors.notFound,
  TOO_MANY_REQUESTS: MESSAGES.apiErrors.tooManyRequests,
  SERVER_ERROR: MESSAGES.apiErrors.serverError,
  INTERNAL_ERROR: MESSAGES.apiErrors.serverError,
  MARSHAL_ERROR: MESSAGES.apiErrors.serverError,
  DATABASE_ERROR: MESSAGES.apiErrors.serverError,
  INVALID_AUTH_PAYLOAD: MESSAGES.apiErrors.sessionExpired,
  TRANSFER_ALREADY_EXISTS: MESSAGES.apiErrors.generalError,
  INVALID_CHUNK_INDEX: MESSAGES.apiErrors.badRequest,
};

const AUTH_ERROR_CODES = new Set([
  'UNAUTHORIZED',
  'SESSION_EXPIRED',
  'INVALID_TOKEN',
  'INVALID_REFRESH_TOKEN',
  'REFRESH_TOKEN_EXPIRED',
  'MISSING_REFRESH_TOKEN',
  'MISSING_AUTHORIZATION',
  'TOKEN_REVOKED',
  'INVALID_AUTH_PAYLOAD',
]);

const SESSION_EXPIRED_CODES = new Set([
  'SESSION_EXPIRED',
  'INVALID_REFRESH_TOKEN',
  'REFRESH_TOKEN_EXPIRED',
  'INVALID_TOKEN',
  'TOKEN_REVOKED',
  'MISSING_REFRESH_TOKEN',
]);

export function isAuthErrorCode(code: string | undefined): boolean {
  return code != null && AUTH_ERROR_CODES.has(code.toUpperCase());
}

function userMessageByCode(code: string | undefined): string | undefined {
  if (!code) return undefined;
  return CODE_TO_USER_MESSAGE[code];
}

function userMessageByStatus(statusCode: number): string | undefined {
  if (statusCode === 400) return MESSAGES.apiErrors.badRequest;
  if (statusCode === 401) return MESSAGES.apiErrors.sessionExpired;
  if (statusCode === 403) return MESSAGES.apiErrors.forbidden;
  if (statusCode === 404) return MESSAGES.apiErrors.notFound;
  if (statusCode === 429) return MESSAGES.apiErrors.tooManyRequests;
  if (statusCode >= 500) return MESSAGES.apiErrors.serverError;
  return undefined;
}

export function parseError(error: unknown): AppError {
  if (!error) {
    return {
      message: MESSAGES.apiErrors.generalError,
      code: ErrorCode.UNKNOWN_ERROR,
    };
  }

  if (error instanceof Error) {
    const appError: AppError = {
      message: error.message,
      originalError: error,
      code: ErrorCode.UNKNOWN_ERROR,
    };
    if ('statusCode' in error) {
      appError.statusCode = error.statusCode as number;
    }
    if (
      'code' in error &&
      typeof (error as { code?: string }).code === 'string'
    ) {
      appError.code = (error as { code: string }).code;
    }
    if (
      error.name === 'TypeError' &&
      (error.message.includes('fetch') || error.message.includes('network'))
    ) {
      appError.code = ErrorCode.NETWORK_ERROR;
      appError.isRetryable = true;
    }
    if (
      error.message.includes('timeout') ||
      error.message.includes('Timeout')
    ) {
      appError.code = ErrorCode.NETWORK_ERROR;
      appError.isRetryable = true;
    }
    return appError;
  }

  if (typeof error === 'string') {
    return {
      message: error,
      code: ErrorCode.UNKNOWN_ERROR,
    };
  }

  if (typeof error === 'object' && error !== null) {
    const o = error as Record<string, unknown>;
    const code = typeof o.code === 'string' ? o.code : undefined;
    const message =
      (typeof o.message === 'string' ? o.message : null) ??
      (typeof o.error === 'string' ? o.error : null) ??
      MESSAGES.apiErrors.generalError;
    const appError: AppError = {
      message,
      code: code || ErrorCode.UNKNOWN_ERROR,
      originalError: error,
    };
    if (typeof o.statusCode === 'number') {
      appError.statusCode = o.statusCode;
    }
    if (
      appError.statusCode != null &&
      appError.statusCode >= HTTP_SERVER_ERROR_THRESHOLD
    ) {
      appError.isRetryable = true;
    }
    return appError;
  }

  return {
    message: MESSAGES.apiErrors.generalError,
    code: ErrorCode.UNKNOWN_ERROR,
    originalError: error,
  };
}

export function getFriendlyErrorMessage(
  error: unknown,
  defaultMessage: string = MESSAGES.apiErrors.generalError
): string {
  const appError = parseError(error);

  if (!navigator.onLine && appError.code === ErrorCode.NETWORK_ERROR) {
    return MESSAGES.apiErrors.offline;
  }

  const byCode = userMessageByCode(appError.code);
  if (byCode) return byCode;

  const byStatus =
    appError.statusCode != null
      ? userMessageByStatus(appError.statusCode)
      : undefined;
  if (byStatus) return byStatus;

  return defaultMessage;
}

export function isUnauthorizedError(error: unknown): boolean {
  const appError = parseError(error);
  return appError.statusCode === 401 || isAuthErrorCode(appError.code);
}

export function isSessionExpiredError(error: unknown): boolean {
  const appError = parseError(error);
  const code = appError.code?.toUpperCase();
  return code != null && SESSION_EXPIRED_CODES.has(code);
}

export async function parseApiError(response: Response): Promise<AppError> {
  const appError: AppError = {
    message:
      response.statusText ||
      MESSAGES.common.http.errorWithStatus(response.status),
    statusCode: response.status,
    code: 'UNKNOWN',
    isRetryable: response.status >= HTTP_SERVER_ERROR_THRESHOLD,
  };

  try {
    const contentType = response.headers.get('content-type');
    if (contentType?.includes('application/json')) {
      const data = (await response.json()) as ApiErrorResponse;
      appError.message = data.message ?? data.error ?? appError.message;
      appError.code = data.code && data.code !== '' ? data.code : 'UNKNOWN';
      if (data.details) {
        appError.originalError = data.details;
      }
    }
  } catch {
    void 0;
  }

  if (response.status === 401) {
    if (appError.code === 'UNKNOWN' || appError.code === 'UNAUTHORIZED') {
      appError.code = 'UNAUTHORIZED';
      appError.message = MESSAGES.apiErrors.sessionExpired;
    }
  } else if (response.status === 403) {
    appError.code = 'UNAUTHORIZED';
    appError.message = MESSAGES.apiErrors.forbidden;
  } else if (response.status === 404) {
    if (appError.code === 'UNKNOWN') appError.code = 'NOT_FOUND';
    appError.message = MESSAGES.apiErrors.notFound;
  } else if (response.status === 429) {
    appError.code = 'TOO_MANY_REQUESTS';
    appError.message = MESSAGES.apiErrors.tooManyRequests;
  } else if (response.status === 503 || response.status === 502) {
    appError.code = 'SERVICE_UNAVAILABLE';
    appError.message = MESSAGES.apiErrors.serviceUnavailable;
    appError.isRetryable = true;
  } else if (response.status >= 500) {
    appError.code =
      appError.code === 'UNKNOWN' ? 'INTERNAL_ERROR' : appError.code;
    appError.message = MESSAGES.apiErrors.serverError;
    appError.isRetryable = true;
  }

  return appError;
}
