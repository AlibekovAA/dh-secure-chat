import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { useToast } from "../../shared/ui/ToastProvider";
import {
  getFriendlyErrorMessage,
  isSessionExpiredError,
} from "../../shared/api/error-handler";
import { apiClient } from "../../shared/api/client";
import { fetchMe, type UserSummary } from "../chat/api";

const AuthScreen = lazy(() => import("./AuthScreen").then((module) => ({ default: module.AuthScreen })));
const ChatScreen = lazy(() => import("./ChatScreen").then((module) => ({ default: module.ChatScreen })));

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
      const json = await apiClient.post<{ token?: string }>("/api/auth/refresh");
      if (!json.token) {
        return null;
      }
      apiClient.setToken(json.token);
      setToken(json.token);
      return json.token;
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    apiClient.setRefreshTokenFn(refreshAccessToken);
    apiClient.setOnTokenExpired(() => {
      setToken(null);
      setProfile(null);
      setSearchResults([]);
    });
  }, [refreshAccessToken]);

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

    apiClient.setToken(token);
    fetchMe()
      .then((data) => {
        setProfile(data);
        localStorage.setItem('userId', data.id);
      })
      .catch((err) => {
        if (isSessionExpiredError(err)) {
          showToast("Сессия истекла. Войдите снова.", "error");
        } else {
          const friendly = getFriendlyErrorMessage(err, "Не удалось получить профиль. Войдите снова.");
          showToast(friendly, "error");
        }
        setToken(null);
        setProfile(null);
        setSearchResults([]);
      });
  }, [token, showToast]);

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
      const params = new URLSearchParams({ username: searchQuery.trim() });
      const users = await apiClient.get<UserSummary[]>(`/api/chat/users?${params.toString()}`);
      const filtered = users.filter((user) => user.id !== profile.id);
      setSearchResults(filtered);
      setHasSearched(true);
    } catch (err) {
      if (isSessionExpiredError(err)) {
        showToast("Сессия истекла. Войдите снова.", "error");
        setToken(null);
        setProfile(null);
        setSearchResults([]);
      } else {
        const friendly = getFriendlyErrorMessage(err, "Ошибка поиска пользователей");
        showToast(friendly, "error");
        setSearchResults([]);
        setHasSearched(true);
      }
    } finally {
      setIsSearching(false);
    }
  }, [token, searchQuery, profile, showToast]);

  const handleLogout = useCallback(async () => {
    try {
      await apiClient.post("/api/auth/logout");
    } catch {
    } finally {
      apiClient.setToken(null);
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

  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center bg-black text-emerald-50">
          <div className="flex flex-col items-center gap-3">
            <div className="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
            <p className="text-xs text-emerald-500/80">Загрузка...</p>
          </div>
        </div>
      }
    >
      {token ? (
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
          onTokenExpired={refreshAccessToken}
        />
      ) : (
        <AuthScreen onAuthenticated={(token) => {
          apiClient.setToken(token);
          setToken(token);
        }} />
      )}
    </Suspense>
  );
}
