import { useEffect, useState } from "react";
import type React from "react";
import type { UserSummary } from "../chat/api";
import { ChatWindow } from "../chat/ChatWindow";

type Profile = {
  id: string;
  username: string;
};

type Props = {
  token: string;
  profile: Profile | null;
  searchQuery: string;
  onSearchQueryChange(value: string): void;
  searchResults: UserSummary[];
  onSearch(): void;
  onLogout(): void;
  onUserSelect(user: UserSummary): void;
  isSearching?: boolean;
  hasSearched?: boolean;
};

const RESULTS_PER_PAGE = 4;

export function ChatScreen({
  token,
  profile,
  searchQuery,
  onSearchQueryChange,
  searchResults,
  onSearch,
  onLogout,
  onUserSelect,
  isSearching = false,
  hasSearched = false,
}: Props) {
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedPeer, setSelectedPeer] = useState<UserSummary | null>(null);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" && !isSearching && searchQuery.trim()) {
      e.preventDefault();
      setCurrentPage(1);
      onSearch();
    }
  };

  const totalPages = Math.ceil(searchResults.length / RESULTS_PER_PAGE);
  const startIndex = (currentPage - 1) * RESULTS_PER_PAGE;
  const endIndex = startIndex + RESULTS_PER_PAGE;
  const paginatedResults = searchResults.slice(startIndex, endIndex);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
  };

  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, searchResults.length]);
  return (
    <div className="min-h-screen flex flex-col bg-black text-emerald-50">
      <header className="flex items-center justify-between px-4 py-3 border-b border-emerald-700/60">
        <div>
          <h1 className="text-xl font-semibold text-emerald-400">dh-secure-chat</h1>
          <p className="text-[11px] text-emerald-500/80">
            Защищённый мессенджер с E2E шифрованием и DH‑обменом ключами
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
                  type="text"
                  value={searchQuery}
                  onChange={e => onSearchQueryChange(e.target.value)}
                  onKeyDown={handleKeyDown}
                  disabled={isSearching}
                  className="w-full rounded-md bg-black border border-emerald-700 pr-24 px-3 py-2 text-sm text-emerald-50 outline-none focus:ring-2 focus:ring-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  placeholder="Имя пользователя..."
                  autoComplete="off"
                />
                <div className="absolute inset-y-0 right-0 flex items-center gap-1 pr-1 w-24">
                  {searchQuery.trim() && !isSearching && (
                    <button
                      type="button"
                      onClick={() => onSearchQueryChange("")}
                      className="flex items-center justify-center w-5 h-5 rounded text-emerald-500 hover:text-emerald-300 hover:bg-emerald-900/40 transition-colors"
                      aria-label="Очистить поиск"
                    >
                      <svg
                        className="w-4 h-4"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M6 18L18 6M6 6l12 12"
                        />
                      </svg>
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={onSearch}
                    disabled={isSearching || !searchQuery.trim()}
                    className="ml-auto rounded-md bg-emerald-500 hover:bg-emerald-400 disabled:bg-emerald-700 disabled:cursor-not-allowed text-xs font-medium px-3 py-1.5 text-black transition-colors min-w-[60px] flex items-center justify-center"
                  >
                    {isSearching ? (
                      <div className="w-3 h-3 border-2 border-black border-t-transparent rounded-full animate-spin" />
                    ) : (
                      "Поиск"
                    )}
                  </button>
                </div>
              </div>
              <div className="h-64 flex flex-col">
                {isSearching ? (
                  <div className="flex items-center justify-center h-full">
                    <div className="w-5 h-5 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
                  </div>
                ) : hasSearched && searchResults.length === 0 && searchQuery.trim() ? (
                  <div className="flex items-center justify-center h-full">
                    <p className="text-emerald-500/80 text-sm">Нет результатов</p>
                  </div>
                ) : searchResults.length === 0 ? (
                  <div className="h-full" />
                ) : (
                  <>
                    <div className="flex-1 space-y-1 text-sm text-emerald-100 min-h-0">
                      {paginatedResults.map(user => (
                        <button
                          key={user.id}
                          type="button"
                          onClick={() => {
                            console.log('User clicked:', user);
                            setSelectedPeer(user);
                            onUserSelect(user);
                            console.log('selectedPeer set to:', user);
                          }}
                          className="w-full text-left rounded-md border border-emerald-700 px-3 py-2 bg-black/60 hover:bg-emerald-900/40 transition-colors active:scale-[0.98]"
                        >
                          <p className="font-medium text-emerald-300">{user.username}</p>
                          <p className="text-[11px] text-emerald-500/80 break-all">{user.id}</p>
                        </button>
                      ))}
                    </div>
                    {totalPages > 1 && (
                      <div className="flex items-center justify-between gap-1 pt-1.5 pb-0.5 border-t border-emerald-700/60 shrink-0">
                        <button
                          type="button"
                          onClick={() => handlePageChange(currentPage - 1)}
                          disabled={currentPage === 1}
                          className="px-1.5 py-0.5 text-xs rounded border border-emerald-700 bg-black/60 text-emerald-400 hover:bg-emerald-900/40 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                          ←
                        </button>
                        <span className="text-xs text-emerald-500/80">
                          {currentPage} / {totalPages}
                        </span>
                        <button
                          type="button"
                          onClick={() => handlePageChange(currentPage + 1)}
                          disabled={currentPage === totalPages}
                          className="px-1.5 py-0.5 text-xs rounded border border-emerald-700 bg-black/60 text-emerald-400 hover:bg-emerald-900/40 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                          →
                        </button>
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          </section>
        </div>
      </main>

      {selectedPeer && profile && (
        <ChatWindow
          token={token}
          peer={selectedPeer}
          myUserId={profile.id}
          onClose={() => setSelectedPeer(null)}
        />
      )}
    </div>
  );
}
