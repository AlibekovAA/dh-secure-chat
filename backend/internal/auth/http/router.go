package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type registerRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	IdentityPubKey string `json:"identity_pub_key"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type Handler struct {
	auth *service.AuthService
	log  *logger.Logger
}

func NewHandler(auth *service.AuthService, log *logger.Logger) http.Handler {
	h := &Handler{auth: auth, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", commonhttp.HealthHandler(log))
	mux.HandleFunc("/api/auth/register", h.register)
	mux.HandleFunc("/api/auth/login", h.login)
	mux.HandleFunc("/api/auth/refresh", h.refresh)
	mux.HandleFunc("/api/auth/logout", h.logout)
	return mux
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		commonhttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warnf("register failed: invalid json: %v", err)
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var pubKey []byte
	if req.IdentityPubKey != "" {
		decoded, err := base64.StdEncoding.DecodeString(req.IdentityPubKey)
		if err != nil {
			h.log.Warnf("register failed: invalid identity_pub_key encoding: %v", err)
			commonhttp.WriteError(w, http.StatusBadRequest, "invalid identity_pub_key encoding")
			return
		}
		pubKey = decoded
	}

	result, err := h.auth.Register(ctx, service.RegisterInput{
		Username:       req.Username,
		Password:       req.Password,
		IdentityPubKey: pubKey,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	setRefreshCookie(w, r, result.RefreshToken, result.RefreshExpiresAt)
	commonhttp.WriteJSON(w, http.StatusCreated, tokenResponse{Token: result.AccessToken})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		commonhttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warnf("login failed: invalid json: %v", err)
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := h.auth.Login(ctx, service.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	setRefreshCookie(w, r, result.RefreshToken, result.RefreshExpiresAt)
	commonhttp.WriteJSON(w, http.StatusOK, tokenResponse{Token: result.AccessToken})
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		commonhttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		commonhttp.WriteError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := h.auth.RefreshAccessToken(ctx, cookie.Value)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) || errors.Is(err, service.ErrRefreshTokenExpired) {
			commonhttp.WriteError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		h.log.Errorf("refresh failed: %v", err)
		commonhttp.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	setRefreshCookie(w, r, result.RefreshToken, result.RefreshExpiresAt)
	commonhttp.WriteJSON(w, http.StatusOK, tokenResponse{Token: result.AccessToken})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		commonhttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := h.auth.RevokeRefreshToken(ctx, cookie.Value); err != nil {
			h.log.Errorf("logout revoke failed: %v", err)
		}
	}

	clearRefreshCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func setRefreshCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	if token == "" {
		return
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/api/auth",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
	}

	http.SetCookie(w, cookie)
}

func clearRefreshCookie(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/auth",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
	}

	http.SetCookie(w, cookie)
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	if vErr, ok := service.AsValidationError(err); ok {
		commonhttp.WriteError(w, http.StatusBadRequest, vErr.Error())
		return
	}

	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		commonhttp.WriteError(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, service.ErrUsernameTaken):
		commonhttp.WriteError(w, http.StatusConflict, "username already taken")
	default:
		commonhttp.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}
