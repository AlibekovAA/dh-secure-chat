import { FormEvent, useState } from "react";
import { login, register } from "./api";
import { useToast } from "../../shared/ui/ToastProvider";

type Mode = "login" | "register";

type Props = {
  onAuthenticated(token: string): void;
};

export function AuthForm({ onAuthenticated }: Props) {
  const { showToast } = useToast();
  const [mode, setMode] = useState<Mode>("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const switchMode = (next: Mode, resetFields: boolean) => {
    setMode(next);
    if (resetFields) {
      setUsername("");
      setPassword("");
      setConfirmPassword("");
      setShowPassword(false);
      setShowConfirmPassword(false);
    }
  };

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!username || !password) {
      showToast("Введите имя пользователя и пароль", "error");
      return;
    }
    if (mode === "register" && password !== confirmPassword) {
      showToast("Пароли не совпадают", "error");
      return;
    }

    setSubmitting(true);

    try {
      if (mode === "login") {
        const result = await login(username, password);
        showToast("Успешный вход", "success");
        onAuthenticated(result.token);
        return;
      }

      await register(username, password);
      showToast("Регистрация прошла успешно. Теперь войдите.", "success");
      switchMode("login", false);
      setConfirmPassword("");
    } catch (err) {
      const raw = err instanceof Error ? err.message : "Ошибка аутентификации";

      let friendly = "Произошла ошибка. Попробуйте ещё раз.";

      if (raw.includes("invalid credentials")) {
        friendly = "Неверное имя пользователя или пароль.";
      } else if (raw.includes("username already taken") || raw.includes("username already exists")) {
        friendly = "Это имя пользователя уже занято.";
      } else if (raw.includes("username must be between")) {
        friendly = "Имя пользователя должно быть от 3 до 32 символов и содержать только буквы, цифры, _ или -.";
      } else if (raw.includes("password must be between")) {
        friendly = "Пароль должен быть от 8 до 72 символов.";
      } else if (raw.includes("username may contain only")) {
        friendly = "Имя пользователя может содержать только буквы, цифры, подчёркивание и дефис.";
      }

      showToast(friendly, "error");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="w-full max-w-sm mx-auto">
      <div className="flex mb-4 rounded-lg overflow-hidden border border-emerald-600">
        <button
          type="button"
          onClick={() => switchMode("login", true)}
          className={`flex-1 py-2 text-sm font-medium ${
            mode === "login"
              ? "bg-emerald-600 text-black"
              : "bg-black text-emerald-400"
          }`}
        >
          Вход
        </button>
        <button
          type="button"
          onClick={() => switchMode("register", true)}
          className={`flex-1 py-2 text-sm font-medium ${
            mode === "register"
              ? "bg-emerald-600 text-black"
              : "bg-black text-emerald-400"
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
          <label className="block text-sm text-emerald-300">Имя пользователя</label>
          <input
            type="text"
            value={username}
            onChange={event => setUsername(event.target.value)}
            className="w-full rounded-md bg-black border border-emerald-700 px-3 py-2 text-sm text-emerald-50 outline-none focus:ring-2 focus:ring-emerald-500"
            autoComplete="username"
            disabled={submitting}
          />
        </div>

        <div className="space-y-1">
          <label className="block text-sm text-emerald-300">Пароль</label>
          <div className="relative">
            <input
              type={showPassword ? "text" : "password"}
              value={password}
              onChange={event => setPassword(event.target.value)}
              className="w-full rounded-md bg-black border border-emerald-700 pr-16 px-3 py-2 text-sm text-emerald-50 outline-none focus:ring-2 focus:ring-emerald-500"
              autoComplete={mode === "login" ? "current-password" : "new-password"}
              disabled={submitting}
            />
            <button
              type="button"
              onClick={() => setShowPassword(current => !current)}
              className="absolute inset-y-0 right-0 flex items-center px-3 text-xs text-emerald-400 hover:text-emerald-200 hover:bg-emerald-900/40 rounded-r-md"
            >
              {showPassword ? "Скрыть" : "Показать"}
            </button>
          </div>
          {mode === "register" && (
            <p className="text-xs text-emerald-500/80">
              Восстановление пароля не предусмотрено. Сохраните пароль в надежном месте.
            </p>
          )}
        </div>

        {mode === "register" && (
          <div className="space-y-1">
            <label className="block text-sm text-emerald-300">Подтвердите пароль</label>
            <div className="relative">
              <input
                type={showConfirmPassword ? "text" : "password"}
                value={confirmPassword}
                onChange={event => setConfirmPassword(event.target.value)}
                className="w-full rounded-md bg-black border border-emerald-700 pr-16 px-3 py-2 text-sm text-emerald-50 outline-none focus:ring-2 focus:ring-emerald-500"
                autoComplete="new-password"
                disabled={submitting}
              />
              <button
                type="button"
                onClick={() => setShowConfirmPassword(current => !current)}
                className="absolute inset-y-0 right-0 flex items-center px-3 text-xs text-emerald-400 hover:text-emerald-200 hover:bg-emerald-900/40 rounded-r-md"
              >
                {showConfirmPassword ? "Скрыть" : "Показать"}
              </button>
            </div>
          </div>
        )}

        <button
          type="submit"
          disabled={submitting}
          className="w-full rounded-md bg-emerald-500 hover:bg-emerald-400 disabled:bg-emerald-700 text-sm font-medium py-2 text-black transition-colors"
        >
          {submitting
            ? mode === "login"
              ? "Входим..."
              : "Регистрируем..."
            : mode === "login"
              ? "Войти"
              : "Зарегистрироваться"}
        </button>
      </form>
    </div>
  );
}
