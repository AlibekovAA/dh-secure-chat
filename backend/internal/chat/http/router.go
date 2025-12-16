package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

type Handler struct {
	chat *service.ChatService
	log  *logger.Logger
}

type meResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type userResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler(chat *service.ChatService, log *logger.Logger) http.Handler {
	h := &Handler{chat: chat, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat/me", h.me)
	mux.HandleFunc("/api/chat/users", h.searchUsers)

	return mux
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwtverify.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	h.log.Infof("chat/me request user_id=%s", claims.UserID)

	user, err := h.chat.GetMe(r.Context(), claims.UserID)
	if err != nil {
		h.log.Errorf("chat/me failed user_id=%s: %v", claims.UserID, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load profile"})
		return
	}

	h.log.Infof("chat/me success user_id=%s", claims.UserID)
	writeJSON(w, http.StatusOK, meResponse{
		ID:       string(user.ID),
		Username: user.Username,
	})
}

func (h *Handler) searchUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	if _, ok := jwtverify.FromContext(r.Context()); !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	query := r.URL.Query().Get("username")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	h.log.Infof("chat/users search query=%q limit=%d", query, limit)

	users, err := h.chat.SearchUsers(r.Context(), query, limit)
	if err != nil {
		h.log.Warnf("chat/users search failed query=%q: %v", query, err)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	resp := toUserResponses(users)
	h.log.Infof("chat/users search success query=%q results=%d", query, len(resp))
	writeJSON(w, http.StatusOK, resp)
}

func toUserResponses(users []userdomain.Summary) []userResponse {
	result := make([]userResponse, 0, len(users))
	for _, u := range users {
		result = append(result, userResponse{
			ID:       string(u.ID),
			Username: u.Username,
		})
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
