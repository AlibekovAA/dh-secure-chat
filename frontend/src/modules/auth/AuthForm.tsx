import { FormEvent, useState, useMemo, useRef, useEffect } from 'react';
import { login, register, updatePublicKey } from '@/modules/auth/api';
import { apiClient } from '@/shared/api/client';
import { useToast } from '@/shared/ui/useToast';
import { getFriendlyErrorMessage } from '@/shared/api/error-handler';
import { validateAuthForm, type ValidationError } from '@/shared/validation';
import { MESSAGES } from '@/shared/messages';
import {
  generateIdentityKeyPair,
  exportPublicKey,
  saveIdentityPrivateKey,
  loadIdentityPrivateKey,
} from '@/shared/crypto/identity';
import { USERNAME_MIN_LENGTH, USERNAME_MAX_LENGTH } from '@/shared/constants';

type Mode = 'login' | 'register';

type PasswordStrength = 'weak' | 'medium' | 'strong' | 'very-strong';

type Props = {
  onAuthenticated(token: string): void;
};

function calculatePasswordStrength(password: string): {
  strength: PasswordStrength;
  score: number;
  feedback: string[];
} {
  if (!password) {
    return { strength: 'weak', score: 0, feedback: [] };
  }

  let score = 0;
  const feedback: string[] = [];

  if (password.length >= 8) score += 1;
  else feedback.push(MESSAGES.auth.passwordStrength.min8);

  if (password.length >= 12) score += 1;

  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) score += 1;
  else if (/[a-zA-Z]/.test(password))
    feedback.push(MESSAGES.auth.passwordStrength.addUppercaseLatin);

  if (/\d/.test(password)) score += 1;
  else feedback.push(MESSAGES.auth.passwordStrength.addDigits);

  if (/[^a-zA-Z0-9]/.test(password)) score += 1;
  else feedback.push(MESSAGES.auth.passwordStrength.addSpecial);

  if (password.length >= 16) score += 1;

  let strength: PasswordStrength;
  if (score <= 2) strength = 'weak';
  else if (score === 3) strength = 'medium';
  else if (score === 4) strength = 'strong';
  else strength = 'very-strong';

  return { strength, score, feedback };
}

export function AuthForm({ onAuthenticated }: Props) {
  const { showToast } = useToast();
  const [mode, setMode] = useState<Mode>('login');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [validationError, setValidationError] =
    useState<ValidationError | null>(null);
  const usernameInputRef = useRef<HTMLInputElement>(null);
  const passwordInputRef = useRef<HTMLInputElement>(null);
  const confirmPasswordInputRef = useRef<HTMLInputElement>(null);

  const passwordStrength = useMemo(
    () => calculatePasswordStrength(password),
    [password]
  );

  useEffect(() => {
    usernameInputRef.current?.focus();
  }, [mode]);

  const trimmedUsername = username.trim();
  const usernameLength = trimmedUsername.length;
  const isUsernameTooShort =
    usernameLength > 0 && usernameLength < USERNAME_MIN_LENGTH;
  const isUsernameTooLong =
    usernameLength > 0 && usernameLength > USERNAME_MAX_LENGTH;
  const usernameCharsValid =
    usernameLength === 0 || /^[a-zA-Z0-9_-]+$/.test(trimmedUsername);
  const usernameStartsOrEndsInvalid =
    usernameLength > 0 &&
    (trimmedUsername[0] === '_' ||
      trimmedUsername[0] === '-' ||
      trimmedUsername[usernameLength - 1] === '_' ||
      trimmedUsername[usernameLength - 1] === '-');
  const isUsernameValid =
    usernameLength >= USERNAME_MIN_LENGTH &&
    usernameLength <= USERNAME_MAX_LENGTH &&
    usernameCharsValid &&
    !usernameStartsOrEndsInvalid;

  const switchMode = (next: Mode, resetFields: boolean) => {
    setMode(next);
    setValidationError(null);
    if (resetFields) {
      setUsername('');
      setPassword('');
      setConfirmPassword('');
      setShowPassword(false);
      setShowConfirmPassword(false);
    }
  };

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();

    const validationError = validateAuthForm(
      mode,
      username,
      password,
      mode === 'register' ? confirmPassword : undefined
    );

    if (validationError) {
      setValidationError(validationError);
      showToast(validationError.message, 'error');
      const fieldRefs = {
        username: usernameInputRef,
        password: passwordInputRef,
        confirmPassword: confirmPasswordInputRef,
      };
      fieldRefs[
        validationError.field as keyof typeof fieldRefs
      ]?.current?.focus();
      return;
    }

    setValidationError(null);
    setSubmitting(true);

    try {
      if (mode === 'login') {
        const result = await login(username, password);
        apiClient.setToken(result.token);
        const existingKey = await loadIdentityPrivateKey();
        if (!existingKey) {
          try {
            const keyPair = await generateIdentityKeyPair();
            await saveIdentityPrivateKey(keyPair.privateKey);
            const pub = await exportPublicKey(keyPair.publicKey);
            await updatePublicKey(pub);
            showToast(MESSAGES.auth.toasts.newKeyCreatedOnDevice, 'success', {
              duration: 3000,
            });
          } catch (keyErr) {
            const friendly = getFriendlyErrorMessage(keyErr);
            showToast(friendly, 'error');
            setSubmitting(false);
            return;
          }
        }
        showToast(MESSAGES.auth.toasts.loginSuccess, 'success', {
          duration: 2000,
        });
        onAuthenticated(result.token);
        return;
      }

      let identityPubKey: string | undefined;
      if (mode === 'register') {
        try {
          const keyPair = await generateIdentityKeyPair();
          identityPubKey = await exportPublicKey(keyPair.publicKey);
          await saveIdentityPrivateKey(keyPair.privateKey);
        } catch (err) {
          const errorMessage =
            err instanceof Error
              ? err.message
              : MESSAGES.auth.errors.keygenDefault;
          showToast(errorMessage, 'error');
          setSubmitting(false);
          return;
        }
      }

      await register(username, password, identityPubKey);
      showToast(MESSAGES.auth.toasts.registerSuccess, 'success', {
        duration: 2000,
      });
      switchMode('login', false);
      setConfirmPassword('');
      setSubmitting(false);
    } catch (err) {
      const friendly = getFriendlyErrorMessage(err);
      showToast(friendly, 'error');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="w-full max-w-sm mx-auto">
      <div className="flex mb-4 overflow-hidden rounded-lg border border-slate-700 bg-[var(--color-surface)]">
        <button
          type="button"
          onClick={() => switchMode('login', true)}
          className={`flex-1 py-2 text-sm font-medium smooth-transition button-press ${
            mode === 'login'
              ? 'bg-emerald-600 text-black'
              : 'text-slate-300 hover:bg-slate-800/80 hover:text-emerald-300'
          }`}
        >
          {MESSAGES.auth.tabs.login}
        </button>
        <button
          type="button"
          onClick={() => switchMode('register', true)}
          className={`flex-1 py-2 text-sm font-medium smooth-transition button-press ${
            mode === 'register'
              ? 'bg-emerald-600 text-black'
              : 'text-slate-300 hover:bg-slate-800/80 hover:text-emerald-300'
          }`}
        >
          {MESSAGES.auth.tabs.register}
        </button>
      </div>

      <form
        onSubmit={handleSubmit}
        className="space-y-4 rounded-lg bg-[var(--color-surface)] px-5 py-4 border border-slate-700/80 border-l-2 border-l-emerald-500/70"
      >
        <div className="space-y-1">
          <label
            htmlFor="auth-username"
            className="block text-sm text-emerald-300"
          >
            {MESSAGES.auth.labels.username}
          </label>
          <input
            ref={usernameInputRef}
            id="auth-username"
            type="text"
            value={username}
            onChange={(event) => {
              setUsername(event.target.value);
              if (validationError?.field === 'username') {
                setValidationError(null);
              }
            }}
            className={`w-full rounded-md bg-[var(--color-surface-raised)] border px-3 py-2 text-sm text-slate-100 outline-none focus:ring-2 ${
              mode === 'register' && username
                ? isUsernameValid
                  ? 'border-emerald-500/70 focus:ring-emerald-500'
                  : 'border-red-500/60 focus:ring-red-500'
                : 'border-slate-700 focus:ring-emerald-500'
            }`}
            autoComplete="username"
            disabled={submitting}
            aria-invalid={validationError?.field === 'username'}
            aria-describedby={
              validationError?.field === 'username'
                ? 'auth-username-error'
                : mode === 'register' && username
                  ? 'auth-username-hint'
                  : undefined
            }
          />
          {validationError?.field === 'username' && (
            <p id="auth-username-error" className="sr-only">
              {validationError.message}
            </p>
          )}
          {mode === 'register' && (
            <div
              className="overflow-hidden transition-all duration-300 ease-out"
              style={{
                maxHeight: username ? '1.5rem' : '0',
                opacity: username ? 1 : 0,
              }}
            >
              <p
                id="auth-username-hint"
                className={`text-xs mt-1 transition-colors ${
                  username && !isUsernameValid
                    ? 'text-red-400'
                    : 'text-emerald-500/70'
                }`}
              >
                {isUsernameTooShort
                  ? MESSAGES.auth.info.usernameMin(USERNAME_MIN_LENGTH)
                  : isUsernameTooLong
                    ? MESSAGES.auth.info.usernameMax(USERNAME_MAX_LENGTH)
                    : !usernameCharsValid
                      ? MESSAGES.auth.info.usernameAllowed
                      : usernameStartsOrEndsInvalid
                        ? MESSAGES.validation.usernameCannotStartOrEnd
                        : MESSAGES.auth.info.usernameOk}
              </p>
            </div>
          )}
        </div>

        <div className="space-y-1">
          <label
            htmlFor="auth-password"
            className="block text-sm text-emerald-300"
          >
            {MESSAGES.auth.labels.password}
          </label>
          <div className="relative">
            <input
              ref={passwordInputRef}
              id="auth-password"
              type={showPassword ? 'text' : 'password'}
              value={password}
              onChange={(event) => {
                setPassword(event.target.value);
                if (validationError?.field === 'password') {
                  setValidationError(null);
                }
              }}
              className="w-full rounded-md bg-[var(--color-surface-raised)] border border-slate-700 pr-16 px-3 py-2 text-sm text-slate-100 outline-none focus:ring-2 focus:ring-emerald-500"
              autoComplete={
                mode === 'login' ? 'current-password' : 'new-password'
              }
              disabled={submitting}
              aria-invalid={validationError?.field === 'password'}
              aria-describedby={
                validationError?.field === 'password'
                  ? 'auth-password-error'
                  : mode === 'register' && password
                    ? 'auth-password-strength'
                    : undefined
              }
            />
            <button
              type="button"
              onClick={() => setShowPassword((current) => !current)}
              aria-pressed={showPassword}
              className="absolute inset-y-0 right-0 flex items-center px-3 text-xs text-emerald-400 hover:text-emerald-200 hover:bg-emerald-900/40 rounded-r-md"
            >
              {showPassword
                ? MESSAGES.auth.actions.hide
                : MESSAGES.auth.actions.show}
            </button>
          </div>
          {validationError?.field === 'password' && (
            <p id="auth-password-error" className="sr-only">
              {validationError.message}
            </p>
          )}
          {mode === 'register' && (
            <div
              className="overflow-hidden transition-all duration-300 ease-out"
              style={{
                maxHeight: password ? '5rem' : '0',
                opacity: password ? 1 : 0,
              }}
            >
              <div id="auth-password-strength" className="space-y-1.5 mt-1">
                <div className="flex gap-1 h-1.5">
                  {[1, 2, 3, 4].map((level) => {
                    const isActive =
                      (passwordStrength.strength === 'weak' && level <= 1) ||
                      (passwordStrength.strength === 'medium' && level <= 2) ||
                      (passwordStrength.strength === 'strong' && level <= 3) ||
                      (passwordStrength.strength === 'very-strong' &&
                        level <= 4);
                    const colorClass =
                      passwordStrength.strength === 'weak'
                        ? 'bg-red-500'
                        : passwordStrength.strength === 'medium'
                          ? 'bg-yellow-500'
                          : passwordStrength.strength === 'strong'
                            ? 'bg-emerald-400'
                            : 'bg-emerald-500';
                    return (
                      <div
                        key={level}
                        className={`flex-1 rounded-full transition-all duration-300 ${
                          isActive ? colorClass : 'bg-emerald-900/30'
                        }`}
                      />
                    );
                  })}
                </div>
                <div className="flex items-center justify-between">
                  <p
                    className={`text-xs font-medium ${
                      passwordStrength.strength === 'weak'
                        ? 'text-red-400'
                        : passwordStrength.strength === 'medium'
                          ? 'text-yellow-400'
                          : passwordStrength.strength === 'strong'
                            ? 'text-emerald-400'
                            : 'text-emerald-300'
                    }`}
                  >
                    {passwordStrength.strength === 'weak'
                      ? MESSAGES.auth.passwordStrength.labels.weak
                      : passwordStrength.strength === 'medium'
                        ? MESSAGES.auth.passwordStrength.labels.medium
                        : passwordStrength.strength === 'strong'
                          ? MESSAGES.auth.passwordStrength.labels.strong
                          : MESSAGES.auth.passwordStrength.labels.veryStrong}
                  </p>
                  {passwordStrength.feedback.length > 0 && (
                    <p className="text-xs text-emerald-500/70">
                      {passwordStrength.feedback[0]}
                    </p>
                  )}
                </div>
              </div>
            </div>
          )}
          {mode === 'register' && (
            <p className="text-xs text-emerald-500/70">
              {MESSAGES.auth.info.noPasswordRecovery}
            </p>
          )}
        </div>

        {mode === 'register' && (
          <div className="space-y-1">
            <label
              htmlFor="auth-confirm-password"
              className="block text-sm text-emerald-300"
            >
              {MESSAGES.auth.labels.confirmPassword}
            </label>
            <div className="relative">
              <input
                ref={confirmPasswordInputRef}
                id="auth-confirm-password"
                type={showConfirmPassword ? 'text' : 'password'}
                value={confirmPassword}
                onChange={(event) => {
                  setConfirmPassword(event.target.value);
                  if (validationError?.field === 'confirmPassword') {
                    setValidationError(null);
                  }
                }}
                className={`w-full rounded-md bg-[var(--color-surface-raised)] border pr-16 px-3 py-2 text-sm text-slate-100 outline-none focus:ring-2 focus:ring-emerald-500 ${
                  confirmPassword && password !== confirmPassword
                    ? 'border-red-500/60 focus:ring-red-500'
                    : confirmPassword && password === confirmPassword
                      ? 'border-emerald-500/60'
                      : 'border-emerald-700'
                }`}
                autoComplete="new-password"
                disabled={submitting}
                aria-invalid={validationError?.field === 'confirmPassword'}
                aria-describedby={
                  validationError?.field === 'confirmPassword'
                    ? 'auth-confirm-password-error'
                    : confirmPassword
                      ? 'auth-confirm-password-hint'
                      : undefined
                }
              />
              <button
                type="button"
                onClick={() => setShowConfirmPassword((current) => !current)}
                aria-pressed={showConfirmPassword}
                className="absolute inset-y-0 right-0 flex items-center px-3 text-xs text-emerald-400 hover:text-emerald-200 hover:bg-emerald-900/40 rounded-r-md"
              >
                {showConfirmPassword
                  ? MESSAGES.auth.actions.hide
                  : MESSAGES.auth.actions.show}
              </button>
            </div>
            {validationError?.field === 'confirmPassword' && (
              <p id="auth-confirm-password-error" className="sr-only">
                {validationError.message}
              </p>
            )}
            <div
              className="overflow-hidden transition-all duration-300 ease-out"
              style={{
                maxHeight: confirmPassword ? '1.5rem' : '0',
                opacity: confirmPassword ? 1 : 0,
              }}
            >
              <p
                id="auth-confirm-password-hint"
                className={`text-xs mt-1 ${
                  password === confirmPassword
                    ? 'text-emerald-400'
                    : 'text-red-400'
                }`}
              >
                {password === confirmPassword
                  ? MESSAGES.auth.info.passwordsMatch
                  : MESSAGES.auth.info.passwordsMismatch}
              </p>
            </div>
          </div>
        )}

        <button
          type="submit"
          disabled={submitting}
          className="w-full rounded-md bg-emerald-500 hover:bg-emerald-400 disabled:bg-slate-700 disabled:text-slate-400 text-sm font-semibold py-2.5 text-slate-950 smooth-transition button-press"
        >
          {submitting
            ? mode === 'login'
              ? MESSAGES.auth.actions.loggingIn
              : MESSAGES.auth.actions.registering
            : mode === 'login'
              ? MESSAGES.auth.actions.login
              : MESSAGES.auth.actions.register}
        </button>
      </form>
    </div>
  );
}
