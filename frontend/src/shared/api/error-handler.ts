import {
  UNAUTHORIZED_MESSAGE,
  SESSION_EXPIRED_ERROR,
} from '@/shared/constants';
import { MESSAGES } from '@/shared/messages';

export type ApiErrorResponse = {
  error: string;
  code?: string;
  details?: Record<string, unknown>;
};

export type AppError = {
  message: string;
  code?: string;
  originalError?: unknown;
  statusCode?: number;
  isRetryable?: boolean;
};

export enum ErrorCode {
  UNAUTHORIZED = 'UNAUTHORIZED',
  SESSION_EXPIRED = 'SESSION_EXPIRED',
  INVALID_CREDENTIALS = 'INVALID_CREDENTIALS',
  USERNAME_TAKEN = 'USERNAME_TAKEN',
  VALIDATION_ERROR = 'VALIDATION_ERROR',
  NETWORK_ERROR = 'NETWORK_ERROR',
  SERVICE_UNAVAILABLE = 'SERVICE_UNAVAILABLE',
  FILE_TOO_LARGE = 'FILE_TOO_LARGE',
  USER_NOT_FOUND = 'USER_NOT_FOUND',
  IDENTITY_KEY_NOT_FOUND = 'IDENTITY_KEY_NOT_FOUND',
  UNKNOWN_ERROR = 'UNKNOWN_ERROR',
}

import { HTTP_SERVER_ERROR_THRESHOLD } from '@/shared/constants';

export type ErrorMapping = {
  pattern: RegExp | string;
  message: string;
};

const ERROR_MAPPINGS: ErrorMapping[] = [
  {
    pattern: /invalid credentials|invalid username or password/i,
    message: MESSAGES.apiErrors.invalidCredentials,
  },
  {
    pattern: /username already taken|username already exists/i,
    message: MESSAGES.apiErrors.usernameTaken,
  },
  {
    pattern: /username must be between/i,
    message: MESSAGES.apiErrors.usernameBetween,
  },
  {
    pattern: /password must be between/i,
    message: MESSAGES.apiErrors.passwordBetween,
  },
  {
    pattern: /username may contain only/i,
    message: MESSAGES.apiErrors.usernameAllowedChars,
  },
  {
    pattern: /password must contain at least/i,
    message: MESSAGES.apiErrors.passwordMustContainLatinAndDigit,
  },
  {
    pattern: /query is empty/i,
    message: MESSAGES.apiErrors.queryEmpty,
  },
  {
    pattern: /query is too long/i,
    message: MESSAGES.apiErrors.queryTooLong,
  },
  {
    pattern: /session_expired|invalid refresh token|refresh token/i,
    message: MESSAGES.apiErrors.sessionExpired,
  },
  {
    pattern: /unauthorized|invalid token|token expired/i,
    message: MESSAGES.apiErrors.sessionExpired,
  },
  {
    pattern: /service temporarily unavailable|circuit breaker/i,
    message: MESSAGES.apiErrors.serviceUnavailable,
  },
  {
    pattern: /file size exceeds maximum/i,
    message: MESSAGES.apiErrors.fileTooLarge,
  },
  {
    pattern: /user not found/i,
    message: MESSAGES.apiErrors.userNotFound,
  },
  {
    pattern: /identity key not found/i,
    message: MESSAGES.apiErrors.identityKeyNotFound,
  },
  {
    pattern: /failed to load profile/i,
    message: MESSAGES.apiErrors.failedToLoadProfile,
  },
  {
    pattern: /search failed/i,
    message: MESSAGES.apiErrors.searchFailed,
  },
  {
    pattern: /failed to get identity key/i,
    message: MESSAGES.apiErrors.failedToGetIdentityKey,
  },
  {
    pattern: /failed to get fingerprint/i,
    message: MESSAGES.apiErrors.failedToGetFingerprint,
  },
  {
    pattern: /network error|failed to fetch|networkrequestfailed/i,
    message: MESSAGES.apiErrors.noServerConnection,
  },
  {
    pattern: /timeout|timed out/i,
    message: MESSAGES.apiErrors.timeout,
  },
  {
    pattern: /websocket|ws connection/i,
    message: MESSAGES.apiErrors.connectionError,
  },
  {
    pattern: new RegExp(`reconnect|${MESSAGES.common.words.reconnect}`, 'i'),
    message: MESSAGES.apiErrors.reconnecting,
  },
  {
    pattern: /encryption|decryption|crypto/i,
    message: MESSAGES.apiErrors.encryptionError,
  },
  {
    pattern: /file.*encrypt|encrypt.*file/i,
    message: MESSAGES.apiErrors.fileEncryptError,
  },
  {
    pattern: /file.*decrypt|decrypt.*file/i,
    message: MESSAGES.apiErrors.fileDecryptError,
  },
];

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

  if (typeof error === 'object' && error !== null && 'error' in error) {
    const apiError = error as ApiErrorResponse;
    const appError: AppError = {
      message: apiError.error || MESSAGES.apiErrors.generalError,
      code: apiError.code || ErrorCode.UNKNOWN_ERROR,
      originalError: error,
    };

    if ('statusCode' in error) {
      appError.statusCode = error.statusCode as number;
    }

    if (apiError.code) {
      appError.code = apiError.code as ErrorCode;
    }

    if (
      appError.statusCode &&
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
  const errorMessage = appError.message.toLowerCase().trim();

  if (!navigator.onLine && appError.code === ErrorCode.NETWORK_ERROR) {
    return MESSAGES.apiErrors.offline;
  }

  for (const mapping of ERROR_MAPPINGS) {
    if (typeof mapping.pattern === 'string') {
      if (errorMessage.includes(mapping.pattern.toLowerCase())) {
        return mapping.message;
      }
    } else {
      if (mapping.pattern.test(appError.message)) {
        return mapping.message;
      }
    }
  }

  if (appError.statusCode) {
    if (appError.statusCode === 400) {
      return MESSAGES.apiErrors.badRequest;
    }
    if (appError.statusCode === 401) {
      if (appError.code === ErrorCode.INVALID_CREDENTIALS) {
        return MESSAGES.apiErrors.invalidCredentials;
      }
      return MESSAGES.apiErrors.sessionExpired;
    }
    if (appError.statusCode === 403) {
      return MESSAGES.apiErrors.forbidden;
    }
    if (appError.statusCode === 404) {
      return MESSAGES.apiErrors.notFound;
    }
    if (appError.statusCode === 429) {
      return MESSAGES.apiErrors.tooManyRequests;
    }
    if (appError.statusCode >= 500) {
      return MESSAGES.apiErrors.serverError;
    }
  }

  const cleanMessage = appError.message
    .replace(/^error\s*\d+:\s*/i, '')
    .replace(
      new RegExp(`^${MESSAGES.common.words.errorLower}\\s*\\d+:\\s*`, 'i'),
      ''
    )
    .trim();

  if (cleanMessage && cleanMessage.length < 100) {
    return cleanMessage;
  }

  return defaultMessage;
}

export function isUnauthorizedError(error: unknown): boolean {
  const appError = parseError(error);
  return (
    appError.code === ErrorCode.UNAUTHORIZED ||
    appError.code === ErrorCode.SESSION_EXPIRED ||
    appError.statusCode === 401 ||
    appError.message.toLowerCase().includes(UNAUTHORIZED_MESSAGE) ||
    appError.message.toLowerCase().includes('invalid token') ||
    appError.message.toLowerCase().includes('token expired')
  );
}

export function isSessionExpiredError(error: unknown): boolean {
  const appError = parseError(error);
  return (
    appError.code === ErrorCode.SESSION_EXPIRED ||
    appError.message.toLowerCase() === SESSION_EXPIRED_ERROR
  );
}

export async function parseApiError(response: Response): Promise<AppError> {
  const appError: AppError = {
    message:
      response.statusText ||
      MESSAGES.common.http.errorWithStatus(response.status),
    statusCode: response.status,
    code: ErrorCode.UNKNOWN_ERROR,
    isRetryable: response.status >= HTTP_SERVER_ERROR_THRESHOLD,
  };

  try {
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      const data = (await response.json()) as ApiErrorResponse;
      appError.message = data.error || appError.message;
      if (data.code) {
        appError.code = data.code as ErrorCode;
      }
      if (data.details) {
        appError.originalError = data.details;
      }
    }
  } catch {
    void 0;
  }

  if (response.status === 401) {
    const errorMessageLower = appError.message.toLowerCase();
    const isInvalidCredentials =
      appError.code === ErrorCode.INVALID_CREDENTIALS ||
      errorMessageLower.includes('invalid credentials') ||
      errorMessageLower.includes('invalid username or password');

    if (isInvalidCredentials) {
      appError.code = ErrorCode.INVALID_CREDENTIALS;
    } else {
      appError.code = ErrorCode.UNAUTHORIZED;
      if (
        !appError.message ||
        appError.message ===
          MESSAGES.common.http.errorWithStatus(response.status) ||
        appError.message === response.statusText
      ) {
        appError.message = MESSAGES.apiErrors.sessionExpired;
      }
    }
  } else if (response.status === 403) {
    appError.code = ErrorCode.UNAUTHORIZED;
    appError.message = MESSAGES.apiErrors.forbidden;
  } else if (response.status === 404) {
    appError.message = MESSAGES.apiErrors.notFound;
  } else if (response.status === 429) {
    appError.message = MESSAGES.apiErrors.tooManyRequests;
  } else if (response.status === 503 || response.status === 502) {
    appError.code = ErrorCode.SERVICE_UNAVAILABLE;
    appError.message = MESSAGES.apiErrors.serviceUnavailable;
    appError.isRetryable = true;
  } else if (response.status >= 500) {
    appError.message = MESSAGES.apiErrors.serverError;
    appError.isRetryable = true;
  }

  return appError;
}
