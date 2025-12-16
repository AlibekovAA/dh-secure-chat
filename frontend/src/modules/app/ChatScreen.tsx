import type { UserSummary } from "../chat/api";

type Profile = {
  id: string;
  username: string;
};

type Props = {
  profile: Profile | null;
  searchQuery: string;
  onSearchQueryChange(value: string): void;
  searchResults: UserSummary[];
  onSearch(): void;
  onLogout(): void;
};

export function ChatScreen({
  profile,
  searchQuery,
  onSearchQueryChange,
  searchResults,
  onSearch,
  onLogout
}: Props) {
  return (
    <div className="min-h-screen flex flex-col bg-black text-emerald-50">
      <header className="flex items-center justify-between px-4 py-3 border-b border-emerald-700/60">
        <div>
          <h1 className="text-xl font-semibold text-emerald-400">dh-secure-chat</h1>
          <p className="text-[11px] text-emerald-500/80">
            Вы вошли. Скоро здесь появится список собеседников и чат.
          </p>
        </div>
        <div className="flex items-center gap-3 text-xs text-emerald-400">
          {profile && (
            <div className="flex items-center gap-2">
              <span className="inline-flex h-2 w-2 rounded-full bg-emerald-400" />
              <span className="font-medium">{profile.username}</span>
            </div>
          )}
          <button
            type="button"
            onClick={onLogout}
            className="text-emerald-400 hover:text-emerald-200 underline underline-offset-4"
          >
            Выйти
          </button>
        </div>
      </header>

      <main className="flex-1 flex items-center justify-center px-4 py-6">
        <div className="w-full max-w-4xl grid gap-6 md:grid-cols-2">
          <section className="rounded-xl bg-black/80 border border-emerald-700 px-5 py-4 text-sm text-emerald-200">
            <h2 className="text-sm font-semibold text-emerald-300 mb-2">Профиль</h2>
            <p className="text-xs text-emerald-500/80 mb-1">Имя: {profile?.username ?? "…"}</p>
            <p className="text-xs text-emerald-500/80 break-all mb-3">ID: {profile?.id ?? "…"}</p>
            <p className="text-xs text-emerald-500/70">
              Выберите собеседника, чтобы начать защищённый диалог.
            </p>
          </section>

          <section className="rounded-xl bg-black/80 border border-emerald-700 px-5 py-4 text-sm text-emerald-200">
            <h2 className="text-sm font-semibold text-emerald-300 mb-2">Найти собеседника</h2>
            <div className="space-y-2">
              <div className="relative">
                <input
                  value={searchQuery}
                  onChange={e => onSearchQueryChange(e.target.value)}
                  className="w-full rounded-md bg-black border border-emerald-700 pr-24 px-3 py-2 text-sm text-emerald-50 outline-none focus:ring-2 focus:ring-emerald-500"
                  placeholder="Имя пользователя…"
                />
                <button
                  type="button"
                  onClick={onSearch}
                  className="absolute inset-y-0 right-0 mx-1 my-1 rounded-md bg-emerald-500 hover:bg-emerald-400 text-xs font-medium px-3 text-black transition-colors"
                >
                  Поиск
                </button>
              </div>
              <div className="space-y-1 text-sm text-emerald-100 max-h-64 overflow-y-auto">
                {searchResults.length === 0 ? (
                  <p className="text-emerald-500/80">Нет результатов</p>
                ) : (
                  searchResults.map(user => (
                    <button
                      key={user.id}
                      type="button"
                      className="w-full text-left rounded-md border border-emerald-700 px-3 py-2 bg-black/60 hover:bg-emerald-900/40 transition-colors"
                    >
                      <p className="font-medium text-emerald-300">{user.username}</p>
                      <p className="text-[11px] text-emerald-500/80 break-all">{user.id}</p>
                    </button>
                  ))
                )}
              </div>
            </div>
          </section>
        </div>
      </main>
    </div>
  );
}
