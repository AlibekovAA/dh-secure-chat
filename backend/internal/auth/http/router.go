package http

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/auth/service"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/config"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
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

func NewHandler(auth *service.AuthService, cfg config.AuthConfig, log *logger.Logger) http.Handler {
	h := &Handler{
		auth: auth,
		log:  log,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", commonhttp.HealthHandler(log))
	mux.HandleFunc("/api/auth/register", commonhttp.RequireMethod(http.MethodPost)(commonhttp.WithTimeout(cfg.RequestTimeout)(h.register)))
	mux.HandleFunc("/api/auth/login", commonhttp.RequireMethod(http.MethodPost)(commonhttp.WithTimeout(cfg.RequestTimeout)(h.login)))
	mux.HandleFunc("/api/auth/refresh", commonhttp.RequireMethod(http.MethodPost)(commonhttp.WithTimeout(cfg.RequestTimeout)(h.refresh)))
	mux.HandleFunc("/api/auth/logout", commonhttp.RequireMethod(http.MethodPost)(commonhttp.WithTimeout(cfg.RequestTimeout)(h.logout)))
	mux.HandleFunc("/api/auth/revoke", commonhttp.RequireMethod(http.MethodPost)(commonhttp.WithTimeout(cfg.RequestTimeout)(h.revoke)))
	return mux
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := commonhttp.DecodeJSON(r, &req); err != nil {
		h.log.Warnf("register failed: invalid json: %v", err)
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx := r.Context()

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
		commonhttp.HandleError(w, r, err, h.log)
		return
	}

	setRefreshCookie(w, r, result.RefreshToken, result.RefreshExpiresAt)
	commonhttp.WriteJSON(w, http.StatusCreated, tokenResponse{Token: result.AccessToken})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := commonhttp.DecodeJSON(r, &req); err != nil {
		h.log.Warnf("login failed: invalid json: %v", err)
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx := r.Context()

	result, err := h.auth.Login(ctx, service.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		commonhttp.HandleError(w, r, err, h.log)
		return
	}

	setRefreshCookie(w, r, result.RefreshToken, result.RefreshExpiresAt)
	commonhttp.WriteJSON(w, http.StatusOK, tokenResponse{Token: result.AccessToken})
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		commonhttp.WriteError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	ctx := r.Context()
	clientIP := commonhttp.GetClientIP(r)

	result, err := h.auth.RefreshAccessToken(ctx, cookie.Value, clientIP)
	if err != nil {
		commonhttp.HandleError(w, r, err, h.log)
		return
	}

	setRefreshCookie(w, r, result.RefreshToken, result.RefreshExpiresAt)
	commonhttp.WriteJSON(w, http.StatusOK, tokenResponse{Token: result.AccessToken})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var claims jwtverify.Claims
	var err error
	if ctxClaims, ok := jwtverify.FromContext(ctx); ok {
		claims = ctxClaims
	} else {
		if tokenString, ok := jwtverify.ExtractTokenFromHeader(r); ok {
			claims, err = h.auth.ParseTokenForRevoke(ctx, tokenString)
			if err != nil {
				claims = jwtverify.Claims{}
			}
		}
	}

	if err == nil && claims.JTI != "" {
		if err := h.auth.RevokeAccessToken(ctx, claims.JTI, claims.UserID); err != nil {
			h.log.Errorf("logout revoke access token failed: %v", err)
		}
	}

	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		if err := h.auth.RevokeRefreshToken(ctx, cookie.Value); err != nil {
			h.log.Errorf("logout revoke refresh token failed: %v", err)
		}
	}

	clearRefreshCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var claims jwtverify.Claims
	var err error
	if ctxClaims, ok := jwtverify.FromContext(ctx); ok {
		claims = ctxClaims
	} else {
		tokenString, ok := jwtverify.ExtractTokenFromHeader(r)
		if !ok {
			commonhttp.WriteError(w, http.StatusUnauthorized, "missing authorization")
			return
		}

		claims, err = h.auth.ParseTokenForRevoke(ctx, tokenString)
		if err != nil {
			commonhttp.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}
	}

	if claims.JTI == "" {
		commonhttp.WriteError(w, http.StatusBadRequest, "token does not have jti")
		return
	}

	if err := h.auth.RevokeAccessToken(ctx, claims.JTI, claims.UserID); err != nil {
		commonhttp.HandleError(w, r, err, h.log)
		return
	}

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
