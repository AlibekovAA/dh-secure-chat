import {
  USERNAME_MIN_LENGTH,
  USERNAME_MAX_LENGTH,
  PASSWORD_MIN_LENGTH,
  PASSWORD_MAX_LENGTH,
  USERNAME_REGEX,
  MAX_MESSAGE_LENGTH,
  MAX_SEARCH_QUERY_LENGTH,
} from '@/shared/constants';

export type ValidationError = {
  field:
    | 'username'
    | 'password'
    | 'confirmPassword'
    | 'message'
    | 'searchQuery';
  message: string;
};

export function validateUsername(username: string): ValidationError | null {
  if (!username || username.trim().length === 0) {
    return { field: 'username', message: 'Введите имя пользователя' };
  }

  if (username.length < USERNAME_MIN_LENGTH) {
    return {
      field: 'username',
      message: `Имя пользователя должно быть минимум ${USERNAME_MIN_LENGTH} символа`,
    };
  }

  if (username.length > USERNAME_MAX_LENGTH) {
    return {
      field: 'username',
      message: `Имя пользователя должно быть максимум ${USERNAME_MAX_LENGTH} символов`,
    };
  }

  if (!USERNAME_REGEX.test(username)) {
    return {
      field: 'username',
      message:
        'Имя пользователя может содержать только латинские буквы, цифры, _ и -',
    };
  }

  const firstChar = username[0];
  const lastChar = username[username.length - 1];
  if (
    firstChar === '_' ||
    firstChar === '-' ||
    lastChar === '_' ||
    lastChar === '-'
  ) {
    return {
      field: 'username',
      message:
        'Имя пользователя не может начинаться или заканчиваться на _ или -',
    };
  }

  return null;
}

export function validatePassword(password: string): ValidationError | null {
  if (!password || password.trim().length === 0) {
    return { field: 'password', message: 'Введите пароль' };
  }

  if (password.length < PASSWORD_MIN_LENGTH) {
    return {
      field: 'password',
      message: `Пароль должен быть минимум ${PASSWORD_MIN_LENGTH} символов`,
    };
  }

  if (password.length > PASSWORD_MAX_LENGTH) {
    return {
      field: 'password',
      message: `Пароль должен быть максимум ${PASSWORD_MAX_LENGTH} символов`,
    };
  }

  if (!/[a-zA-Z]/.test(password)) {
    return {
      field: 'password',
      message: 'Пароль должен содержать хотя бы одну букву',
    };
  }

  if (!/\d/.test(password)) {
    return {
      field: 'password',
      message: 'Пароль должен содержать хотя бы одну цифру',
    };
  }

  return null;
}

export function validateConfirmPassword(
  password: string,
  confirmPassword: string
): ValidationError | null {
  if (!confirmPassword || confirmPassword.trim().length === 0) {
    return { field: 'confirmPassword', message: 'Подтвердите пароль' };
  }

  if (password !== confirmPassword) {
    return {
      field: 'confirmPassword',
      message: 'Пароли не совпадают',
    };
  }

  return null;
}

export function validateMessage(message: string): ValidationError | null {
  if (!message || message.trim().length === 0) {
    return { field: 'message', message: 'Введите сообщение' };
  }

  if (message.length > MAX_MESSAGE_LENGTH) {
    return {
      field: 'message',
      message: `Сообщение слишком длинное (максимум ${MAX_MESSAGE_LENGTH} символов)`,
    };
  }

  return null;
}

export function validateSearchQuery(query: string): ValidationError | null {
  if (!query || query.trim().length === 0) {
    return { field: 'searchQuery', message: 'Введите поисковый запрос' };
  }

  if (query.trim().length > MAX_SEARCH_QUERY_LENGTH) {
    return {
      field: 'searchQuery',
      message: `Поисковый запрос слишком длинный (максимум ${MAX_SEARCH_QUERY_LENGTH} символов)`,
    };
  }

  return null;
}

export function validateAuthForm(
  mode: 'login' | 'register',
  username: string,
  password: string,
  confirmPassword?: string
): ValidationError | null {
  const usernameError = validateUsername(username);
  if (usernameError) return usernameError;

  const passwordError = validatePassword(password);
  if (passwordError) return passwordError;

  if (mode === 'register' && confirmPassword !== undefined) {
    const confirmError = validateConfirmPassword(password, confirmPassword);
    if (confirmError) return confirmError;
  }

  return null;
}
