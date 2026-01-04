import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useToast } from "../../shared/ui/ToastProvider";
import {
  getFriendlyErrorMessage,
  isSessionExpiredError,
} from "../../shared/api/error-handler";
import { apiClient } from "../../shared/api/client";
import { fetchMe, type UserSummary } from "../chat/api";
import { LoadingSpinner } from "../../shared/ui/LoadingSpinner";

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

  const resetAuthState = useCallback(() => {
    setToken(null);
    setProfile(null);
    setSearchResults([]);
  }, []);

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
    apiClient.setOnTokenExpired(resetAuthState);

    let cancelled = false;

    const init = async () => {
      try {
        const newToken = await refreshAccessToken();
        if (cancelled) {
          return;
        }
        if (!newToken) {
          resetAuthState();
        }
      } catch (err) {
        if (cancelled) {
          return;
        }
        resetAuthState();
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
  }, [refreshAccessToken, resetAuthState]);

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
        resetAuthState();
      });
  }, [token, showToast, resetAuthState]);

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
        resetAuthState();
      } else {
        const friendly = getFriendlyErrorMessage(err, "Ошибка поиска пользователей");
        showToast(friendly, "error");
        setSearchResults([]);
        setHasSearched(true);
      }
    } finally {
      setIsSearching(false);
    }
  }, [token, searchQuery, profile, showToast, resetAuthState]);

  const handleLogout = useCallback(async () => {
    try {
      await apiClient.post("/api/auth/logout");
    } catch {
    } finally {
      apiClient.setToken(null);
      resetAuthState();
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
  }, [resetAuthState]);

  const handleAuthenticated = useCallback((newToken: string) => {
    apiClient.setToken(newToken);
    setToken(newToken);
  }, []);

  const chatScreenProps = useMemo(
    () => ({
      token: token!,
      profile: profile!,
      searchQuery,
      onSearchQueryChange: setSearchQuery,
      searchResults,
      onSearch: handleSearch,
      isSearching,
      hasSearched,
      onLogout: handleLogout,
      onTokenExpired: refreshAccessToken,
    }),
    [token, profile, searchQuery, searchResults, handleSearch, isSearching, hasSearched, handleLogout, refreshAccessToken]
  );

  if (isInitializing) {
    return <LoadingSpinner />;
  }

  return (
    <Suspense fallback={<LoadingSpinner />}>
      {token ? (
        <ChatScreen {...chatScreenProps} />
      ) : (
        <AuthScreen onAuthenticated={handleAuthenticated} />
      )}
    </Suspense>
  );
}
