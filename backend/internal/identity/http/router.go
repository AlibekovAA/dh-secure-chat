package http

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
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
	mux.HandleFunc("/api/identity/users/", commonhttp.RequireMethod(http.MethodGet)(commonhttp.WithTimeout(5*time.Second)(h.handleIdentityRoutes)))

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
	h.handleIdentityRequest(w, r, "/key", "get-public-key", func(ctx context.Context, userID string) (map[string]string, error) {
		pubKey, err := h.identity.GetPublicKey(ctx, userID)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"public_key": base64.StdEncoding.EncodeToString(pubKey),
		}, nil
	})
}

func (h *Handler) getFingerprint(w http.ResponseWriter, r *http.Request) {
	h.handleIdentityRequest(w, r, "/fingerprint", "get-fingerprint", func(ctx context.Context, userID string) (map[string]string, error) {
		fingerprint, err := h.identity.GetFingerprint(ctx, userID)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"fingerprint": fingerprint,
		}, nil
	})
}

func (h *Handler) handleIdentityRequest(
	w http.ResponseWriter,
	r *http.Request,
	suffix string,
	operation string,
	handler func(context.Context, string) (map[string]string, error),
) {
	urlPath := r.URL.Path
	if !strings.HasSuffix(urlPath, suffix) {
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid path")
		return
	}

	userID, ok := commonhttp.ExtractUserIDFromPath(urlPath)
	if !ok || userID == "" {
		h.log.Warnf("identity/%s failed: empty user_id", operation)
		commonhttp.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	userID = strings.TrimSuffix(userID, suffix)
	if err := commonhttp.ValidateUUID(userID); err != nil {
		h.log.Warnf("identity/%s failed: invalid user_id format user_id=%s", operation, userID)
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid user_id format (must be UUID)")
		return
	}

	ctx := r.Context()

	result, err := handler(ctx, userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrIdentityKeyNotFound) {
			h.log.Warnf("identity/%s failed user_id=%s: not found", operation, userID)
			commonhttp.WriteError(w, http.StatusNotFound, "identity key not found")
			return
		}
		h.log.Errorf("identity/%s failed user_id=%s: %v", operation, userID, err)
		commonhttp.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.log.Infof("identity/%s success user_id=%s", operation, userID)
	commonhttp.WriteJSON(w, http.StatusOK, result)
}
