package http

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
)

type Handler struct {
	identity *service.IdentityService
	log      *logger.Logger
}

func NewHandler(identity *service.IdentityService, log *logger.Logger) http.Handler {
	h := &Handler{
		identity: identity,
		log:      log,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/identity/users/", h.handleIdentityRoutes)

	return mux
}

func (h *Handler) handleIdentityRoutes(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	if strings.HasSuffix(urlPath, "/fingerprint") {
		h.getFingerprint(w, r)
		return
	}
	if strings.HasSuffix(urlPath, "/key") {
		h.getPublicKey(w, r)
		return
	}
	commonhttp.WriteError(w, http.StatusBadRequest, "invalid path")
}

func (h *Handler) getPublicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		commonhttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if _, ok := jwtverify.FromContext(r.Context()); !ok {
		commonhttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	urlPath := r.URL.Path
	if !strings.HasSuffix(urlPath, "/key") {
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid path")
		return
	}

	userID := strings.TrimPrefix(urlPath, "/api/identity/users/")
	userID = strings.TrimSuffix(userID, "/key")
	if userID == "" {
		h.log.Warnf("identity/get-public-key failed: empty user_id")
		commonhttp.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	pubKey, err := h.identity.GetPublicKey(ctx, userID)
	if err != nil {
		if errors.Is(err, service.ErrIdentityKeyNotFound) {
			h.log.Warnf("identity/get-public-key failed user_id=%s: not found", userID)
			commonhttp.WriteError(w, http.StatusNotFound, "identity key not found")
			return
		}
		h.log.Errorf("identity/get-public-key failed user_id=%s: %v", userID, err)
		commonhttp.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.log.Infof("identity/get-public-key success user_id=%s", userID)
	commonhttp.WriteJSON(w, http.StatusOK, map[string]string{
		"public_key": base64.StdEncoding.EncodeToString(pubKey),
	})
}

func (h *Handler) getFingerprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		commonhttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if _, ok := jwtverify.FromContext(r.Context()); !ok {
		commonhttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	urlPath := r.URL.Path
	if !strings.HasSuffix(urlPath, "/fingerprint") {
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid path")
		return
	}

	userID := strings.TrimPrefix(urlPath, "/api/identity/users/")
	userID = strings.TrimSuffix(userID, "/fingerprint")
	if userID == "" {
		h.log.Warnf("identity/get-fingerprint failed: empty user_id")
		commonhttp.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	fingerprint, err := h.identity.GetFingerprint(ctx, userID)
	if err != nil {
		if errors.Is(err, service.ErrIdentityKeyNotFound) {
			h.log.Warnf("identity/get-fingerprint failed user_id=%s: not found", userID)
			commonhttp.WriteError(w, http.StatusNotFound, "identity key not found")
			return
		}
		h.log.Errorf("identity/get-fingerprint failed user_id=%s: %v", userID, err)
		commonhttp.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.log.Infof("identity/get-fingerprint success user_id=%s", userID)
	commonhttp.WriteJSON(w, http.StatusOK, map[string]string{
		"fingerprint": fingerprint,
	})
}
