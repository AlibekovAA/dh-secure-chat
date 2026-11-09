import { AuthForm } from '@/modules/auth/AuthForm';
import { MESSAGES } from '@/shared/messages';

type Props = {
  onAuthenticated(token: string): void;
};

export function AuthScreen({ onAuthenticated }: Props) {
  return (
    <div className="app-min-height flex items-center justify-center bg-[var(--color-canvas)] text-slate-100 px-4 py-8">
      <div className="w-full max-w-md space-y-4">
        <header className="text-center space-y-2">
          <h1 className="text-2xl font-semibold text-emerald-300 tracking-tight font-mono">
            dh-secure-chat
          </h1>
          <p className="text-sm text-slate-400 leading-relaxed">
            {MESSAGES.app.authScreen.description}
          </p>
        </header>
        <AuthForm onAuthenticated={onAuthenticated} />
      </div>
    </div>
  );
}
