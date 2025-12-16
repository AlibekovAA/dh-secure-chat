import { AuthForm } from "../auth/AuthForm";

type Props = {
  onAuthenticated(token: string): void;
};

export function AuthScreen({ onAuthenticated }: Props) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-black text-emerald-50 px-4">
      <div className="w-full max-w-md space-y-4">
        <header className="text-center space-y-1">
          <h1 className="text-2xl font-semibold text-emerald-400">dh-secure-chat</h1>
          <p className="text-xs text-emerald-500/80">
            End-to-end зашифрованный чат 1-на-1. Начните с регистрации или входа.
          </p>
        </header>
        <AuthForm onAuthenticated={onAuthenticated} />
      </div>
    </div>
  );
}
