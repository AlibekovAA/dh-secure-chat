import { UNAUTHORIZED_MESSAGE, SESSION_EXPIRED_ERROR } from '../constants';

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

import { HTTP_SERVER_ERROR_THRESHOLD } from '../constants';

export type ErrorMapping = {
  pattern: RegExp | string;
  message: string;
};

const ERROR_MAPPINGS: ErrorMapping[] = [
  {
    pattern: /invalid credentials|invalid username or password/i,
    message: 'Неверное имя пользователя или пароль.',
  },
  {
    pattern: /username already taken|username already exists/i,
    message: 'Это имя пользователя уже занято.',
  },
  {
    pattern: /username must be between/i,
    message: 'Имя пользователя должно быть от 3 до 32 символов.',
  },
  {
    pattern: /password must be between/i,
    message: 'Пароль должен быть от 8 до 72 символов.',
  },
  {
    pattern: /username may contain only/i,
    message:
      'Имя пользователя может содержать только буквы, цифры, подчёркивание и дефис.',
  },
  {
    pattern: /password must contain at least/i,
    message: 'Пароль должен содержать хотя бы одну букву и одну цифру.',
  },
  {
    pattern: /query is empty/i,
    message: 'Поисковый запрос не может быть пустым.',
  },
  {
    pattern: /query is too long/i,
    message: 'Поисковый запрос слишком длинный.',
  },
  {
    pattern: /session_expired|invalid refresh token|refresh token/i,
    message: 'Сессия истекла. Войдите снова.',
  },
  {
    pattern: /unauthorized|invalid token|token expired/i,
    message: 'Сессия истекла. Войдите снова.',
  },
  {
    pattern: /service temporarily unavailable|circuit breaker/i,
    message: 'Сервис временно недоступен. Попробуйте позже.',
  },
  {
    pattern: /file size exceeds maximum/i,
    message: 'Файл слишком большой. Максимальный размер: 50MB.',
  },
  {
    pattern: /user not found/i,
    message: 'Пользователь не найден.',
  },
  {
    pattern: /identity key not found/i,
    message: 'Ключ идентификации не найден.',
  },
  {
    pattern: /failed to load profile/i,
    message: 'Не удалось загрузить профиль.',
  },
  {
    pattern: /search failed/i,
    message: 'Ошибка поиска пользователей.',
  },
  {
    pattern: /failed to get identity key/i,
    message: 'Не удалось получить ключ идентификации.',
  },
  {
    pattern: /failed to get fingerprint/i,
    message: 'Не удалось получить fingerprint.',
  },
];

export function parseError(error: unknown): AppError {
  if (!error) {
    return {
      message: 'Произошла ошибка. Попробуйте ещё раз.',
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
      message: apiError.error || 'Произошла ошибка. Попробуйте ещё раз.',
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
    message: 'Произошла ошибка. Попробуйте ещё раз.',
    code: ErrorCode.UNKNOWN_ERROR,
    originalError: error,
  };
}

export function getFriendlyErrorMessage(
  error: unknown,
  defaultMessage = 'Произошла ошибка. Попробуйте ещё раз.',
): string {
  const appError = parseError(error);
  const errorMessage = appError.message.toLowerCase().trim();

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

  return appError.message || defaultMessage;
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
    message: `Ошибка ${response.status}: ${response.statusText}`,
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
  } catch {}

  if (response.status === 401) {
    appError.code = ErrorCode.UNAUTHORIZED;
  } else if (response.status === 503 || response.status === 502) {
    appError.code = ErrorCode.SERVICE_UNAVAILABLE;
    appError.isRetryable = true;
  }

  return appError;
}
