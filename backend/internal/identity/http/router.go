package http

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
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
	mux.HandleFunc("/api/identity/users/", commonhttp.RequireMethod(http.MethodGet)(commonhttp.WithTimeout(constants.IdentityRequestTimeout)(h.handleIdentityRoutes)))

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

	userID, err := commonhttp.ExtractAndValidateUserID(urlPath, suffix)
	if err != nil {
		if err == commonerrors.ErrEmptyUUID {
			h.log.WithFields(r.Context(), logger.Fields{
				"operation": operation,
				"action":    "identity_empty_user_id",
			}).Warn("identity request failed: empty user_id")
			commonhttp.WriteError(w, http.StatusBadRequest, "user_id is required")
			return
		}
		h.log.WithFields(r.Context(), logger.Fields{
			"operation": operation,
			"user_id":   userID,
			"action":    "identity_invalid_user_id_format",
		}).Warn("identity request failed: invalid user_id format")
		commonhttp.WriteError(w, http.StatusBadRequest, "invalid user_id format (must be UUID)")
		return
	}

	ctx := r.Context()

	result, err := handler(ctx, userID)
	if err != nil {
		commonhttp.HandleError(w, r, err, h.log)
		return
	}

	h.log.WithFields(ctx, logger.Fields{
		"operation": operation,
		"user_id":   userID,
		"action":    "identity_success",
	}).Info("identity request success")
	commonhttp.WriteJSON(w, http.StatusOK, result)
}
