import { FormEvent, useState, useMemo, useRef, useEffect } from 'react';
import { login, register, updatePublicKey } from '@/modules/auth/api';
import { apiClient } from '@/shared/api/client';
import { useToast } from '@/shared/ui/useToast';
import { getFriendlyErrorMessage } from '@/shared/api/error-handler';
import { validateAuthForm } from '@/shared/validation';
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
  const usernameInputRef = useRef<HTMLInputElement>(null);

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
      showToast(validationError.message, 'error');
      return;
    }

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
      <div className="flex mb-4 rounded-lg overflow-hidden border border-emerald-600">
        <button
          type="button"
          onClick={() => switchMode('login', true)}
          className={`flex-1 py-2 text-sm font-medium smooth-transition button-press ${
            mode === 'login'
              ? 'bg-emerald-600 text-black'
              : 'bg-black text-emerald-400 hover:bg-emerald-900/40'
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
              : 'bg-black text-emerald-400 hover:bg-emerald-900/40'
          }`}
        >
          {MESSAGES.auth.tabs.register}
        </button>
      </div>

      <form
        onSubmit={handleSubmit}
        className="space-y-4 rounded-xl bg-black/80 px-5 py-4 border border-emerald-700"
      >
        <div className="space-y-1">
          <label className="block text-sm text-emerald-300">
            {MESSAGES.auth.labels.username}
          </label>
          <input
            ref={usernameInputRef}
            type="text"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            className={`w-full rounded-md bg-black border px-3 py-2 text-sm text-emerald-50 outline-none focus:ring-2 ${
              mode === 'register' && username
                ? isUsernameValid
                  ? 'border-emerald-500/70 focus:ring-emerald-500'
                  : 'border-red-500/60 focus:ring-red-500'
                : 'border-emerald-700 focus:ring-emerald-500'
            }`}
            autoComplete="username"
            disabled={submitting}
          />
          {mode === 'register' && (
            <div
              className="overflow-hidden transition-all duration-300 ease-out"
              style={{
                maxHeight: username ? '1.5rem' : '0',
                opacity: username ? 1 : 0,
              }}
            >
              <p
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
          <label className="block text-sm text-emerald-300">
            {MESSAGES.auth.labels.password}
          </label>
          <div className="relative">
            <input
              type={showPassword ? 'text' : 'password'}
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              className="w-full rounded-md bg-black border border-emerald-700 pr-16 px-3 py-2 text-sm text-emerald-50 outline-none focus:ring-2 focus:ring-emerald-500"
              autoComplete={
                mode === 'login' ? 'current-password' : 'new-password'
              }
              disabled={submitting}
            />
            <button
              type="button"
              onClick={() => setShowPassword((current) => !current)}
              className="absolute inset-y-0 right-0 flex items-center px-3 text-xs text-emerald-400 hover:text-emerald-200 hover:bg-emerald-900/40 rounded-r-md"
            >
              {showPassword
                ? MESSAGES.auth.actions.hide
                : MESSAGES.auth.actions.show}
            </button>
          </div>
          {mode === 'register' && (
            <div
              className="overflow-hidden transition-all duration-300 ease-out"
              style={{
                maxHeight: password ? '5rem' : '0',
                opacity: password ? 1 : 0,
              }}
            >
              <div className="space-y-1.5 mt-1">
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
            <label className="block text-sm text-emerald-300">
              {MESSAGES.auth.labels.confirmPassword}
            </label>
            <div className="relative">
              <input
                type={showConfirmPassword ? 'text' : 'password'}
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                className={`w-full rounded-md bg-black border pr-16 px-3 py-2 text-sm text-emerald-50 outline-none focus:ring-2 focus:ring-emerald-500 ${
                  confirmPassword && password !== confirmPassword
                    ? 'border-red-500/60 focus:ring-red-500'
                    : confirmPassword && password === confirmPassword
                      ? 'border-emerald-500/60'
                      : 'border-emerald-700'
                }`}
                autoComplete="new-password"
                disabled={submitting}
              />
              <button
                type="button"
                onClick={() => setShowConfirmPassword((current) => !current)}
                className="absolute inset-y-0 right-0 flex items-center px-3 text-xs text-emerald-400 hover:text-emerald-200 hover:bg-emerald-900/40 rounded-r-md"
              >
                {showConfirmPassword
                  ? MESSAGES.auth.actions.hide
                  : MESSAGES.auth.actions.show}
              </button>
            </div>
            <div
              className="overflow-hidden transition-all duration-300 ease-out"
              style={{
                maxHeight: confirmPassword ? '1.5rem' : '0',
                opacity: confirmPassword ? 1 : 0,
              }}
            >
              <p
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
          className="w-full rounded-md bg-emerald-500 hover:bg-emerald-400 disabled:bg-emerald-700 text-sm font-medium py-2 text-black smooth-transition button-press glow-emerald-hover"
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
