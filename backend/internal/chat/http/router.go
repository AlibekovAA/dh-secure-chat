package http

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	gorillaWS "github.com/gorilla/websocket"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/websocket"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	identityrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/repository"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

type Handler struct {
	chat      *service.ChatService
	hub       *websocket.Hub
	jwtSecret string
	upgrader  gorillaWS.Upgrader
	log       *logger.Logger
}

type meResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type userResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func NewHandler(chat *service.ChatService, hub *websocket.Hub, jwtSecret string, log *logger.Logger) http.Handler {
	h := &Handler{
		chat:      chat,
		hub:       hub,
		jwtSecret: jwtSecret,
		upgrader: gorillaWS.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				host := r.Host
				if host == "" {
					host = r.URL.Host
				}
				return origin == "http://"+host || origin == "https://"+host
			},
		},
		log: log,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat/me", h.me)
	mux.HandleFunc("/api/chat/users", h.searchUsers)
	mux.HandleFunc("/api/chat/users/", h.getIdentityKey)
	mux.HandleFunc("/ws/", h.handleWebSocket)

	return mux
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwtverify.FromContext(r.Context())
	if !ok {
		commonhttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := h.chat.GetMe(ctx, claims.UserID)
	if err != nil {
		h.log.Errorf("chat/me failed user_id=%s: %v", claims.UserID, err)
		commonhttp.WriteError(w, http.StatusInternalServerError, "failed to load profile")
		return
	}
	h.log.Infof("chat/me success user_id=%s", claims.UserID)
	commonhttp.WriteJSON(w, http.StatusOK, meResponse{
		ID:       string(user.ID),
		Username: user.Username,
	})
}

func (h *Handler) searchUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		commonhttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if _, ok := jwtverify.FromContext(r.Context()); !ok {
		commonhttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
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

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	users, err := h.chat.SearchUsers(ctx, query, limit)
	if err != nil {
		if errors.Is(err, service.ErrEmptyQuery) {
			h.log.Warnf("chat/users search failed query=%q: empty query", query)
			commonhttp.WriteError(w, http.StatusBadRequest, "query is empty")
		} else {
			h.log.Errorf("chat/users search failed query=%q limit=%d: %v", query, limit, err)
			commonhttp.WriteError(w, http.StatusInternalServerError, "failed to search users")
		}
		return
	}

	h.log.Infof("chat/users search success query=%q limit=%d results=%d", query, limit, len(users))
	resp := toUserResponses(users)
	commonhttp.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) getIdentityKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		commonhttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if _, ok := jwtverify.FromContext(r.Context()); !ok {
		commonhttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	urlPath := r.URL.Path
	if !strings.HasSuffix(urlPath, "/identity-key") {
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid path")
		return
	}

	userID := strings.TrimPrefix(urlPath, "/api/chat/users/")
	userID = strings.TrimSuffix(userID, "/identity-key")
	if userID == "" {
		commonhttp.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	pubKey, err := h.chat.GetIdentityKey(ctx, userID)
	if err != nil {
		if errors.Is(err, identityrepo.ErrIdentityKeyNotFound) {
			h.log.Warnf("chat/identity-key failed user_id=%s: not found", userID)
			commonhttp.WriteError(w, http.StatusNotFound, "identity key not found")
		} else {
			h.log.Errorf("chat/identity-key failed user_id=%s: %v", userID, err)
			commonhttp.WriteError(w, http.StatusInternalServerError, "failed to get identity key")
		}
		return
	}

	h.log.Infof("chat/identity-key success user_id=%s", userID)
	commonhttp.WriteJSON(w, http.StatusOK, map[string]string{
		"public_key": base64.StdEncoding.EncodeToString(pubKey),
	})
}

func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Errorf("websocket upgrade failed: %v", err)
		return
	}

	client := websocket.NewUnauthenticatedClient(h.hub, conn, h.jwtSecret, h.log)
	client.Start()
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
