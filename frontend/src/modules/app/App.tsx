import { useEffect, useState } from "react";
import { useToast } from "../../shared/ui/ToastProvider";
import { fetchMe, searchUsers, type UserSummary } from "../chat/api";
import { AuthScreen } from "./AuthScreen";
import { ChatScreen } from "./ChatScreen";

export function App() {
  const [token, setToken] = useState<string | null>(null);
  const [profile, setProfile] = useState<{ id: string; username: string } | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<UserSummary[]>([]);
  const { showToast } = useToast();

  useEffect(() => {
    if (!token) {
      setProfile(null);
      setSearchResults([]);
      return;
    }

    fetchMe(token)
      .then(data => setProfile(data))
      .catch(() => {
        showToast("Не удалось получить профиль. Войдите снова.", "error");
        setToken(null);
      });
  }, [token, showToast]);

  const handleSearch = async () => {
    if (!token || !searchQuery.trim()) {
      return;
    }
    try {
      const users = await searchUsers(searchQuery.trim(), token);
      const filtered = profile
        ? users.filter(user => user.id !== profile.id)
        : users;
      setSearchResults(filtered);
    } catch (err) {
      showToast("Ошибка поиска пользователей", "error");
    }
  };

  return token ? (
    <ChatScreen
      profile={profile}
      searchQuery={searchQuery}
      onSearchQueryChange={setSearchQuery}
      searchResults={searchResults}
      onSearch={handleSearch}
      onLogout={() => setToken(null)}
    />
  ) : (
    <AuthScreen onAuthenticated={setToken} />
  );
}
