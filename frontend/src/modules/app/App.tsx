import { useState } from "react";
import { AuthForm } from "../auth/AuthForm";
import { ToastProvider } from "../../shared/ui/ToastProvider";

export function App() {
  const [token, setToken] = useState<string | null>(null);

  return (
    <ToastProvider>
      {!token ? (
        <div className="min-h-screen flex items-center justify-center bg-black text-emerald-50 px-4">
          <div className="w-full max-w-md space-y-4">
            <header className="text-center space-y-1">
              <h1 className="text-2xl font-semibold text-emerald-400">dh-secure-chat</h1>
              <p className="text-xs text-emerald-500/80">
                End-to-end зашифрованный чат 1-на-1. Начните с регистрации или входа.
              </p>
            </header>
            <AuthForm onAuthenticated={setToken} />
          </div>
        </div>
      ) : (
        <div className="min-h-screen flex items-center justify-center bg-black text-emerald-50 px-4">
          <div className="w-full max-w-md space-y-4">
            <header className="flex items-center justify-between">
              <div>
                <h1 className="text-2xl font-semibold text-emerald-400">dh-secure-chat</h1>
                <p className="text-xs text-emerald-500/80">
                  Вы вошли. Далее будет реализован список собеседников и чат.
                </p>
              </div>
              <button
                type="button"
                onClick={() => setToken(null)}
                className="text-xs text-emerald-400 hover:text-emerald-200 underline underline-offset-4"
              >
                Выйти
              </button>
            </header>
            <div className="rounded-xl bg-black/80 border border-emerald-700 px-5 py-4 text-sm text-emerald-200">
              <p>JWT-токен хранится пока только в памяти этого компонента.</p>
              <p className="mt-1">Следующий шаг: хранилище сессии и интеграция с WebSocket-чатом.</p>
            </div>
          </div>
        </div>
      )}
    </ToastProvider>
  );
}
