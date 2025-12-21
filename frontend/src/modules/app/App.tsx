import { useCallback, useEffect, useState } from "react";
import { useToast } from "../../shared/ui/ToastProvider";
import { fetchMe, searchUsers, SESSION_EXPIRED_ERROR, UNAUTHORIZED_MESSAGE, type UserSummary } from "../chat/api";
import { AuthScreen } from "./AuthScreen";
import { ChatScreen } from "./ChatScreen";

async function fetchWithRetry<T>(
  fetcher: (token: string) => Promise<T>,
  token: string,
  refreshFn: () => Promise<string | null>,
): Promise<T> {
  try {
    return await fetcher(token);
  } catch (err) {
    const isUnauthorized =
      err instanceof Error && err.message.toLowerCase().includes(UNAUTHORIZED_MESSAGE);

    if (!isUnauthorized) {
      throw err;
    }

    const newToken = await refreshFn();
    if (!newToken) {
      throw new Error(SESSION_EXPIRED_ERROR);
    }

    return await fetcher(newToken);
  }
}

export function App() {
  const [token, setToken] = useState<string | null>(null);
  const [profile, setProfile] = useState<{ id: string; username: string } | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<UserSummary[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const [isInitializing, setIsInitializing] = useState(true);
  const { showToast } = useToast();

  const refreshAccessToken = useCallback(async (): Promise<string | null> => {
    try {
      const res = await fetch("/api/auth/refresh", {
        method: "POST",
        credentials: "include",
      });

      if (!res.ok) {
        return null;
      }

      const json = (await res.json()) as { token?: string; error?: string };
      if (!json.token) {
        return null;
      }

      setToken(json.token);
      return json.token;
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    const init = async () => {
      try {
        const newToken = await refreshAccessToken();
        if (cancelled) {
          return;
        }
        if (!newToken) {
          setToken(null);
          setProfile(null);
          setSearchResults([]);
        }
      } catch (err) {
        if (cancelled) {
          return;
        }
        setToken(null);
        setProfile(null);
        setSearchResults([]);
      } finally {
        if (!cancelled) {
          setIsInitializing(false);
        }
      }
    };

    void init();

    return () => {
      cancelled = true;
    };
  }, [refreshAccessToken]);

  useEffect(() => {
    const handleBeforeUnload = () => {
      try {
        import("../../shared/storage/indexeddb").then(({ clearAllKeys }) => {
          clearAllKeys().catch(() => {});
        });
      } catch {
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, []);

  useEffect(() => {
    if (!token) {
      setProfile(null);
      setSearchResults([]);
      return;
    }

    fetchWithRetry((t) => fetchMe(t), token, refreshAccessToken)
      .then((data) => {
        setProfile(data);
        localStorage.setItem('userId', data.id);
      })
      .catch((err) => {
        if (err instanceof Error && err.message === SESSION_EXPIRED_ERROR) {
          showToast("Сессия истекла. Войдите снова.", "error");
        } else {
          showToast("Не удалось получить профиль. Войдите снова.", "error");
        }
        setToken(null);
        setProfile(null);
        setSearchResults([]);
      });
  }, [token, refreshAccessToken, showToast]);

  useEffect(() => {
    if (searchQuery.trim() === "") {
      setSearchResults([]);
      setHasSearched(false);
    }
  }, [searchQuery]);

  const handleSearch = useCallback(async () => {
    if (!token || !searchQuery.trim() || !profile) {
      return;
    }

    setIsSearching(true);
    try {
      const users = await fetchWithRetry(
        (t) => searchUsers(searchQuery.trim(), t),
        token,
        refreshAccessToken,
      );
      const filtered = users.filter((user) => user.id !== profile.id);
      setSearchResults(filtered);
      setHasSearched(true);
    } catch (err) {
      if (err instanceof Error && err.message === SESSION_EXPIRED_ERROR) {
        showToast("Сессия истекла. Войдите снова.", "error");
        setToken(null);
        setProfile(null);
        setSearchResults([]);
      } else {
        showToast("Ошибка поиска пользователей", "error");
        setSearchResults([]);
        setHasSearched(true);
      }
    } finally {
      setIsSearching(false);
    }
  }, [token, searchQuery, profile, refreshAccessToken, showToast]);

  const handleLogout = useCallback(async () => {
    try {
      await fetch("/api/auth/logout", {
        method: "POST",
        credentials: "include",
      });
    } catch {
    } finally {
      setToken(null);
      setProfile(null);
      setSearchResults([]);
      setSearchQuery("");
      setHasSearched(false);

      try {
        const { removeToken } = await import("../../shared/storage/token");
        removeToken();

        const { clearAllKeys } = await import("../../shared/storage/indexeddb");
        await clearAllKeys();

        localStorage.removeItem('userId');
      } catch {
      }
    }
  }, []);

  if (isInitializing) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-black text-emerald-50">
        <div className="flex flex-col items-center gap-3">
          <div className="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
          <p className="text-xs text-emerald-500/80">Загрузка...</p>
        </div>
      </div>
    );
  }

  return token ? (
    <ChatScreen
      token={token}
      profile={profile}
      searchQuery={searchQuery}
      onSearchQueryChange={setSearchQuery}
      searchResults={searchResults}
      onSearch={handleSearch}
      isSearching={isSearching}
      hasSearched={hasSearched}
      onLogout={handleLogout}
      onUserSelect={() => {}}
    />
  ) : (
    <AuthScreen onAuthenticated={setToken} />
  );
}
