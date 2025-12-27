package http

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	gorillaWS "github.com/gorilla/websocket"
	"github.com/jackc/pgx/v4/pgxpool"

	authrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/repository"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/websocket"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/dto"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type Handler struct {
	chat      *service.ChatService
	hub       websocket.HubInterface
	jwtSecret string
	upgrader  gorillaWS.Upgrader
	log       *logger.Logger
	cfg       config.ChatConfig
	pool      *pgxpool.Pool
}

type userResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func NewHandler(chat *service.ChatService, hub websocket.HubInterface, cfg config.ChatConfig, log *logger.Logger, pool *pgxpool.Pool) http.Handler {
	h := &Handler{
		chat:      chat,
		hub:       hub,
		jwtSecret: cfg.JWTSecret,
		cfg:       cfg,
		pool:      pool,
		upgrader: gorillaWS.Upgrader{
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			EnableCompression: true,
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
	mux.HandleFunc("/api/chat/me", commonhttp.WithTimeout(cfg.RequestTimeout)(h.me))
	mux.HandleFunc("/api/chat/users", commonhttp.RequireMethod(http.MethodGet)(commonhttp.WithTimeout(cfg.SearchTimeout)(h.searchUsers)))
	mux.HandleFunc("/api/chat/users/", commonhttp.RequireMethod(http.MethodGet)(commonhttp.WithTimeout(cfg.RequestTimeout)(h.getIdentityKey)))
	mux.HandleFunc("/ws/", h.handleWebSocket)

	return mux
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwtverify.FromContext(r.Context())
	if !ok {
		commonhttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx := r.Context()

	user, err := h.chat.GetMe(ctx, claims.UserID)
	if err != nil {
		commonhttp.HandleError(w, r, err, h.log)
		return
	}
	h.log.WithFields(ctx, logger.Fields{
		"user_id": claims.UserID,
		"action":  "chat_me_success",
	}).Info("chat/me success")
	commonhttp.WriteJSON(w, http.StatusOK, userResponse{
		ID:       user.ID,
		Username: user.Username,
	})
}

func (h *Handler) searchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("username")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	ctx := r.Context()

	users, err := h.chat.SearchUsers(ctx, query, limit)
	if err != nil {
		commonhttp.HandleError(w, r, err, h.log)
		return
	}

	h.log.WithFields(ctx, logger.Fields{
		"query":   query,
		"limit":   limit,
		"results": len(users),
		"action":  "chat_users_search_success",
	}).Info("chat/users search success")
	resp := toUserResponses(users)
	commonhttp.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) getIdentityKey(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	if !strings.HasSuffix(urlPath, "/identity-key") {
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid path")
		return
	}

	ctx := r.Context()
	userID, err := commonhttp.ExtractAndValidateUserID(urlPath, "/identity-key")
	if err != nil {
		if err == commonerrors.ErrEmptyUUID {
			commonhttp.WriteError(w, http.StatusBadRequest, "user_id is required")
			return
		}
		h.log.WithFields(ctx, logger.Fields{
			"user_id": userID,
			"action":  "chat_identity_key_invalid_format",
		}).Warn("chat/identity-key failed: invalid user_id format")
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid user_id format (must be UUID)")
		return
	}

	pubKey, err := h.chat.GetIdentityKey(ctx, userID)
	if err != nil {
		commonhttp.HandleError(w, r, err, h.log)
		return
	}

	h.log.WithFields(ctx, logger.Fields{
		"user_id": userID,
		"action":  "chat_identity_key_success",
	}).Info("chat/identity-key success")
	commonhttp.WriteJSON(w, http.StatusOK, map[string]string{
		"public_key": base64.StdEncoding.EncodeToString(pubKey),
	})
}

func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	revokedTokenRepo := authrepo.NewPgRevokedTokenRepository(h.pool)
	var claims jwtverify.Claims
	var authenticated bool

	if tokenString, ok := jwtverify.ExtractTokenFromHeader(r); ok {
		parsedClaims, err := jwtverify.ParseToken(tokenString, []byte(h.jwtSecret))
		if err == nil {
			if parsedClaims.JTI != "" {
				revoked, err := revokedTokenRepo.IsRevoked(ctx, parsedClaims.JTI)
				if err == nil && !revoked {
					claims = parsedClaims
					authenticated = true
				}
			} else {
				claims = parsedClaims
				authenticated = true
			}
		}
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithFields(ctx, logger.Fields{
			"action": "ws_upgrade_failed",
		}).Errorf("websocket upgrade failed: %v", err)
		return
	}

	var client *websocket.Client
	if authenticated {
		client = websocket.NewAuthenticatedClient(
			h.hub,
			conn,
			claims,
			h.log,
			h.cfg.WebSocketWriteWait,
			h.cfg.WebSocketPongWait,
			h.cfg.WebSocketPingPeriod,
			h.cfg.WebSocketMaxMsgSize,
			h.cfg.WebSocketSendBufSize,
		)
		h.hub.Register(client)
		h.log.WithFields(ctx, logger.Fields{
			"user_id":  claims.UserID,
			"username": claims.Username,
			"action":   "ws_authenticated_via_header",
		}).Info("websocket client authenticated via Authorization header")
	} else {
		client = websocket.NewUnauthenticatedClient(
			h.hub,
			conn,
			h.jwtSecret,
			h.log,
			revokedTokenRepo,
			h.cfg.WebSocketWriteWait,
			h.cfg.WebSocketPongWait,
			h.cfg.WebSocketPingPeriod,
			h.cfg.WebSocketMaxMsgSize,
			h.cfg.WebSocketAuthTimeout,
			h.cfg.WebSocketSendBufSize,
		)
	}
	client.Start()
}

func toUserResponses(users []dto.UserSummary) []userResponse {
	result := make([]userResponse, 0, len(users))
	for _, u := range users {
		result = append(result, userResponse{
			ID:       u.ID,
			Username: u.Username,
		})
	}
	return result
}
