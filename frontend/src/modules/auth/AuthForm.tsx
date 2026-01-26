import { FormEvent, useState, useMemo, useRef, useEffect } from 'react';
import { login, register } from '@/modules/auth/api';
import { useToast } from '@/shared/ui/useToast';
import { getFriendlyErrorMessage } from '@/shared/api/error-handler';
import { validateAuthForm } from '@/shared/validation';
import {
  generateIdentityKeyPair,
  exportPublicKey,
  saveIdentityPrivateKey,
} from '@/shared/crypto/identity';

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
  else feedback.push('Минимум 8 символов');

  if (password.length >= 12) score += 1;

  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) score += 1;
  else if (/[a-zA-Z]/.test(password)) feedback.push('Добавьте заглавные буквы');

  if (/\d/.test(password)) score += 1;
  else feedback.push('Добавьте цифры');

  if (/[^a-zA-Z0-9]/.test(password)) score += 1;
  else feedback.push('Добавьте спецсимволы (!@#$%...)');

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
        showToast('Успешный вход', 'success', { duration: 2000 });
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
              : 'Ошибка генерации ключей. Попробуйте ещё раз.';
          showToast(errorMessage, 'error');
          setSubmitting(false);
          return;
        }
      }

      await register(username, password, identityPubKey);
      showToast('Регистрация прошла успешно.', 'success', { duration: 2000 });
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
          Вход
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
          Регистрация
        </button>
      </div>

      <form
        onSubmit={handleSubmit}
        className="space-y-4 rounded-xl bg-black/80 px-5 py-4 border border-emerald-700"
      >
        <div className="space-y-1">
          <label className="block text-sm text-emerald-300">
            Имя пользователя
          </label>
          <input
            ref={usernameInputRef}
            type="text"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            className="w-full rounded-md bg-black border border-emerald-700 px-3 py-2 text-sm text-emerald-50 outline-none focus:ring-2 focus:ring-emerald-500"
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
              <p className="text-xs text-emerald-500/70 mt-1">
                {username.length < 3
                  ? 'Минимум 3 символа'
                  : username.length > 32
                    ? 'Максимум 32 символа'
                    : /^[a-zA-Z0-9_-]+$/.test(username)
                      ? '✓ Корректное имя'
                      : 'Только буквы, цифры, _ и -'}
              </p>
            </div>
          )}
        </div>

        <div className="space-y-1">
          <label className="block text-sm text-emerald-300">Пароль</label>
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
              {showPassword ? 'Скрыть' : 'Показать'}
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
                      ? 'Слабый'
                      : passwordStrength.strength === 'medium'
                        ? 'Средний'
                        : passwordStrength.strength === 'strong'
                          ? 'Сильный'
                          : 'Очень сильный'}
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
              Восстановление пароля невозможно. Сохраните пароль в надёжном
              месте.
            </p>
          )}
        </div>

        {mode === 'register' && (
          <div className="space-y-1">
            <label className="block text-sm text-emerald-300">
              Подтвердите пароль
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
                {showConfirmPassword ? 'Скрыть' : 'Показать'}
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
                  ? 'Пароли совпадают'
                  : 'Пароли не совпадают'}
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
              ? 'Входим...'
              : 'Регистрируем...'
            : mode === 'login'
              ? 'Войти'
              : 'Зарегистрироваться'}
        </button>
      </form>
    </div>
  );
}
