export type ApiErrorResponse = {
  error: string;
};

export const UNAUTHORIZED_MESSAGE = 'unauthorized';
export const SESSION_EXPIRED_ERROR = 'session_expired';

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
    pattern: /rate limit exceeded/i,
    message: 'Превышен лимит запросов. Попробуйте позже.',
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

export function getFriendlyErrorMessage(
  error: unknown,
  defaultMessage = 'Произошла ошибка. Попробуйте ещё раз.',
): string {
  if (!error) {
    return defaultMessage;
  }

  let errorMessage = '';
  if (error instanceof Error) {
    errorMessage = error.message;
  } else if (typeof error === 'string') {
    errorMessage = error;
  } else if (
    typeof error === 'object' &&
    error !== null &&
    'error' in error &&
    typeof error.error === 'string'
  ) {
    errorMessage = error.error;
  } else {
    return defaultMessage;
  }

  const normalizedMessage = errorMessage.toLowerCase().trim();

  for (const mapping of ERROR_MAPPINGS) {
    if (typeof mapping.pattern === 'string') {
      if (normalizedMessage.includes(mapping.pattern.toLowerCase())) {
        return mapping.message;
      }
    } else {
      if (mapping.pattern.test(errorMessage)) {
        return mapping.message;
      }
    }
  }

  return errorMessage || defaultMessage;
}

export function isUnauthorizedError(error: unknown): boolean {
  if (!error) {
    return false;
  }

  let errorMessage = '';
  if (error instanceof Error) {
    errorMessage = error.message;
  } else if (typeof error === 'string') {
    errorMessage = error;
  } else {
    return false;
  }

  const normalizedMessage = errorMessage.toLowerCase().trim();
  return (
    normalizedMessage.includes(UNAUTHORIZED_MESSAGE) ||
    normalizedMessage === SESSION_EXPIRED_ERROR ||
    normalizedMessage.includes('invalid token') ||
    normalizedMessage.includes('token expired')
  );
}

export function isSessionExpiredError(error: unknown): boolean {
  if (!error) {
    return false;
  }

  let errorMessage = '';
  if (error instanceof Error) {
    errorMessage = error.message;
  } else if (typeof error === 'string') {
    errorMessage = error;
  } else {
    return false;
  }

  const normalizedMessage = errorMessage.toLowerCase().trim();
  return normalizedMessage === SESSION_EXPIRED_ERROR;
}

export async function parseApiError(response: Response): Promise<string> {
  try {
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      const data = (await response.json()) as ApiErrorResponse;
      return data.error || `Ошибка ${response.status}: ${response.statusText}`;
    }
  } catch {}

  return `Ошибка ${response.status}: ${response.statusText}`;
}
