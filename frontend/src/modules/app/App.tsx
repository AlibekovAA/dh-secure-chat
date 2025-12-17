import { useCallback, useEffect, useState } from "react";
import { useToast } from "../../shared/ui/ToastProvider";
import { loadToken, removeToken, saveToken } from "../../shared/storage/token";
import { fetchMe, searchUsers, type UserSummary } from "../chat/api";
import { AuthScreen } from "./AuthScreen";
import { ChatScreen } from "./ChatScreen";

export function App() {
  const [token, setToken] = useState<string | null>(loadToken());
  const [profile, setProfile] = useState<{ id: string; username: string } | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<UserSummary[]>([]);
  const { showToast } = useToast();


  useEffect(() => {
    if (!token) {
      setProfile(null);
      setSearchResults([]);
      removeToken();
      return;
    }

    saveToken(token);
    fetchMe(token)
      .then(data => setProfile(data))
      .catch(() => {
        showToast("Не удалось получить профиль. Войдите снова.", "error");
        setToken(null);
        removeToken();
      });
  }, [token, showToast]);

  const [isSearching, setIsSearching] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);

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
      const users = await searchUsers(searchQuery.trim(), token);
      const filtered = users.filter(user => user.id !== profile.id);
      setSearchResults(filtered);
      setHasSearched(true);
    } catch (err) {
      showToast("Ошибка поиска пользователей", "error");
      setSearchResults([]);
      setHasSearched(true);
    } finally {
      setIsSearching(false);
    }
  }, [token, searchQuery, profile, showToast]);

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
      onLogout={() => {
        setToken(null);
        removeToken();
      }}
      onUserSelect={(user) => {
        // Handled in ChatScreen
      }}
    />
  ) : (
    <AuthScreen onAuthenticated={setToken} />
  );
}
