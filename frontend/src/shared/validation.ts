import {
  USERNAME_MIN_LENGTH,
  USERNAME_MAX_LENGTH,
  PASSWORD_MIN_LENGTH,
  PASSWORD_MAX_LENGTH,
  USERNAME_REGEX,
  MAX_MESSAGE_LENGTH,
  MAX_SEARCH_QUERY_LENGTH,
} from '@/shared/constants';
import { MESSAGES } from '@/shared/messages';

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
    return { field: 'username', message: MESSAGES.validation.usernameRequired };
  }

  if (username.length < USERNAME_MIN_LENGTH) {
    return {
      field: 'username',
      message: MESSAGES.validation.usernameMinLength(USERNAME_MIN_LENGTH),
    };
  }

  if (username.length > USERNAME_MAX_LENGTH) {
    return {
      field: 'username',
      message: MESSAGES.validation.usernameMaxLength(USERNAME_MAX_LENGTH),
    };
  }

  if (!USERNAME_REGEX.test(username)) {
    return {
      field: 'username',
      message: MESSAGES.validation.usernameAllowedChars,
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
      message: MESSAGES.validation.usernameCannotStartOrEnd,
    };
  }

  return null;
}

export function validatePassword(password: string): ValidationError | null {
  if (!password || password.trim().length === 0) {
    return { field: 'password', message: MESSAGES.validation.passwordRequired };
  }

  if (password.length < PASSWORD_MIN_LENGTH) {
    return {
      field: 'password',
      message: MESSAGES.validation.passwordMinLength(PASSWORD_MIN_LENGTH),
    };
  }

  if (password.length > PASSWORD_MAX_LENGTH) {
    return {
      field: 'password',
      message: MESSAGES.validation.passwordMaxLength(PASSWORD_MAX_LENGTH),
    };
  }

  if (!/[a-zA-Z]/.test(password)) {
    return {
      field: 'password',
      message: MESSAGES.validation.passwordMustContainLatinLetter,
    };
  }

  if (!/\d/.test(password)) {
    return {
      field: 'password',
      message: MESSAGES.validation.passwordMustContainDigit,
    };
  }

  return null;
}

export function validateConfirmPassword(
  password: string,
  confirmPassword: string
): ValidationError | null {
  if (!confirmPassword || confirmPassword.trim().length === 0) {
    return {
      field: 'confirmPassword',
      message: MESSAGES.validation.confirmPasswordRequired,
    };
  }

  if (password !== confirmPassword) {
    return {
      field: 'confirmPassword',
      message: MESSAGES.validation.passwordsDoNotMatch,
    };
  }

  return null;
}

export function validateMessage(message: string): ValidationError | null {
  if (!message || message.trim().length === 0) {
    return { field: 'message', message: MESSAGES.validation.messageRequired };
  }

  if (message.length > MAX_MESSAGE_LENGTH) {
    return {
      field: 'message',
      message: MESSAGES.validation.messageTooLong(MAX_MESSAGE_LENGTH),
    };
  }

  return null;
}

export function validateSearchQuery(query: string): ValidationError | null {
  if (!query || query.trim().length === 0) {
    return {
      field: 'searchQuery',
      message: MESSAGES.validation.searchQueryRequired,
    };
  }

  if (query.trim().length > MAX_SEARCH_QUERY_LENGTH) {
    return {
      field: 'searchQuery',
      message: MESSAGES.validation.searchQueryTooLong(MAX_SEARCH_QUERY_LENGTH),
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
