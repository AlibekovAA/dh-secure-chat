import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { useToast } from '@/shared/ui/useToast';
import {
  getFriendlyErrorMessage,
  isSessionExpiredError,
} from '@/shared/api/error-handler';
import { validateSearchQuery } from '@/shared/validation';
import { apiClient } from '@/shared/api/client';
import { fetchMe, type UserSummary } from '@/modules/chat/api';
import { LoadingSpinner } from '@/shared/ui/LoadingSpinner';
import { retryImport } from '@/shared/utils/retry-import';
import {
  attachTokenToClient,
  clearSessionStorageSideEffects,
  handleSessionExpired,
} from '@/shared/api/session';
import { MESSAGES } from '@/shared/messages';

const AuthScreen = lazy(() =>
  retryImport(() =>
    import('@/modules/app/AuthScreen').then((module) => ({
      default: module.AuthScreen,
    }))
  )
);
const ChatScreen = lazy(() =>
  retryImport(() =>
    import('@/modules/app/ChatScreen').then((module) => ({
      default: module.ChatScreen,
    }))
  )
);

export function App() {
  const [token, setToken] = useState<string | null>(null);
  const [profile, setProfile] = useState<{
    id: string;
    username: string;
  } | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
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
      const json = await apiClient.post<{ token?: string }>(
        '/api/auth/refresh'
      );
      if (!json.token) {
        return null;
      }
      attachTokenToClient(json.token);
      setToken(json.token);
      return json.token;
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    apiClient.setRefreshTokenFn(refreshAccessToken);
    apiClient.setOnTokenExpired(() => {
      void handleSessionExpired(
        null,
        { showToast, resetAuthState },
        { silent: true }
      );
    });

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
      } catch {
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
  }, [refreshAccessToken, resetAuthState, showToast]);

  useEffect(() => {
    const handleBeforeUnload = () => {
      try {
        import('@/shared/storage/indexeddb').then(({ clearAllKeys }) => {
          clearAllKeys().catch(() => {
            void 0;
          });
        });
      } catch {
        void 0;
      }
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload);
    };
  }, []);

  useEffect(() => {
    if (!token) {
      setProfile(null);
      setSearchResults([]);
      return;
    }

    attachTokenToClient(token);
    fetchMe()
      .then((data) => {
        setProfile(data);
        localStorage.setItem('userId', data.id);
      })
      .catch((err) => {
        if (
          err &&
          typeof err === 'object' &&
          'silent' in err &&
          (err as { silent?: boolean }).silent
        ) {
          resetAuthState();
          return;
        }
        if (isSessionExpiredError(err)) {
          void handleSessionExpired(err, { showToast, resetAuthState });
        } else {
          const friendly = getFriendlyErrorMessage(
            err,
            MESSAGES.app.errors.failedToLoadProfile
          );
          if (friendly) {
            showToast(friendly, 'error');
          }
          resetAuthState();
        }
      });
  }, [token, showToast, resetAuthState]);

  useEffect(() => {
    if (searchQuery.trim() === '') {
      setSearchResults([]);
      setHasSearched(false);
    }
  }, [searchQuery]);

  const handleSearch = useCallback(async () => {
    if (!token || !profile) {
      return;
    }

    const validationError = validateSearchQuery(searchQuery);
    if (validationError) {
      showToast(validationError.message, 'error');
      return;
    }

    setIsSearching(true);
    try {
      const params = new URLSearchParams({ username: searchQuery.trim() });
      const users = await apiClient.get<UserSummary[]>(
        `/api/chat/users?${params.toString()}`
      );
      const filtered = users.filter((user) => user.id !== profile.id);
      setSearchResults(filtered);
      setHasSearched(true);
    } catch (err) {
      if (
        err &&
        typeof err === 'object' &&
        'silent' in err &&
        (err as { silent?: boolean }).silent
      ) {
        resetAuthState();
        return;
      }
      if (isSessionExpiredError(err)) {
        void handleSessionExpired(err, { showToast, resetAuthState });
      } else {
        const friendly = getFriendlyErrorMessage(
          err,
          MESSAGES.apiErrors.searchFailed
        );
        showToast(friendly, 'error');
        setSearchResults([]);
        setHasSearched(true);
      }
    } finally {
      setIsSearching(false);
    }
  }, [token, searchQuery, profile, showToast, resetAuthState]);

  const handleLogout = useCallback(async () => {
    try {
      await apiClient.post('/api/auth/logout');
    } catch {
      void 0;
    } finally {
      attachTokenToClient(null);
      resetAuthState();
      setSearchQuery('');
      setHasSearched(false);

      await clearSessionStorageSideEffects();
    }
  }, [resetAuthState]);

  const handleAuthenticated = useCallback((newToken: string) => {
    attachTokenToClient(newToken);
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
    [
      token,
      profile,
      searchQuery,
      searchResults,
      handleSearch,
      isSearching,
      hasSearched,
      handleLogout,
      refreshAccessToken,
    ]
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
