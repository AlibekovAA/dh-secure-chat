package http

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/identity/service"
)

type updatePublicKeyRequest struct {
	PublicKey string `json:"public_key"`
}

type updatePublicKeyResponse struct {
	Success bool `json:"success"`
}

type Handler struct {
	identity *service.IdentityService
	log      *logger.Logger
}

func NewHandler(identity service.Service, log *logger.Logger) http.Handler {
	h := &Handler{
		identity: identity.(*service.IdentityService),
		log:      log,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/identity/update-public-key", commonhttp.RequireMethod(http.MethodPost)(commonhttp.WithTimeout(constants.IdentityRequestTimeout)(h.handleUpdatePublicKey)))
	mux.HandleFunc("/api/identity/users/", commonhttp.RequireMethod(http.MethodGet)(commonhttp.WithTimeout(constants.IdentityRequestTimeout)(h.handleIdentityRoutes)))

	return mux
}

func (h *Handler) handleUpdatePublicKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwtverify.FromContext(r.Context())
	if !ok || claims.UserID == "" {
		h.log.WithFields(r.Context(), logger.Fields{
			"action": "update_public_key_unauthorized",
		}).Warn("update public key: missing or invalid auth")
		commonhttp.WriteErrorEnvelope(w, http.StatusUnauthorized, commonhttp.CodeMissingAuthorization, "unauthorized", nil, "")
		return
	}

	var req updatePublicKeyRequest
	if err := commonhttp.DecodeJSON(r, &req); err != nil {
		h.log.WithFields(r.Context(), logger.Fields{
			"user_id": claims.UserID,
			"action":  "update_public_key_invalid_json",
		}).Warnf("update public key failed: invalid json: %v", err)
		commonhttp.WriteErrorEnvelope(w, http.StatusBadRequest, commonhttp.CodeInvalidJSON, "invalid json", nil, "")
		return
	}

	if req.PublicKey == "" {
		h.log.WithFields(r.Context(), logger.Fields{
			"user_id": claims.UserID,
			"action":  "update_public_key_empty",
		}).Warn("update public key failed: empty public_key")
		commonhttp.WriteErrorEnvelope(w, http.StatusBadRequest, commonhttp.CodeBadRequest, "public_key is required", nil, "")
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		h.log.WithFields(r.Context(), logger.Fields{
			"user_id": claims.UserID,
			"action":  "update_public_key_invalid_encoding",
		}).Warnf("update public key failed: invalid base64: %v", err)
		commonhttp.WriteErrorEnvelope(w, http.StatusBadRequest, commonhttp.CodeInvalidIdentityPubKeyEnc, "invalid public_key encoding", nil, "")
		return
	}

	err = h.identity.UpdatePublicKey(r.Context(), claims.UserID, decoded)
	if err != nil {
		commonhttp.HandleError(w, r, err, h.log)
		return
	}

	h.log.WithFields(r.Context(), logger.Fields{
		"user_id": claims.UserID,
		"action":  "update_public_key_success",
	}).Info("update public key success")
	commonhttp.WriteJSON(w, http.StatusOK, updatePublicKeyResponse{Success: true})
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
	commonhttp.WriteErrorEnvelope(w, http.StatusBadRequest, commonhttp.CodeInvalidPath, "invalid path", nil, "")
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
		commonhttp.WriteErrorEnvelope(w, http.StatusBadRequest, commonhttp.CodeInvalidPath, "invalid path", nil, "")
		return
	}

	userID, err := commonhttp.ExtractAndValidateUserID(urlPath, suffix)
	if err != nil {
		if err == commonerrors.ErrEmptyUUID {
			h.log.WithFields(r.Context(), logger.Fields{
				"operation": operation,
				"action":    "identity_empty_user_id",
			}).Warn("identity request failed: empty user_id")
			commonhttp.WriteErrorEnvelope(w, http.StatusBadRequest, commonhttp.CodeUserIDRequired, "user_id is required", nil, "")
			return
		}
		h.log.WithFields(r.Context(), logger.Fields{
			"operation": operation,
			"user_id":   userID,
			"action":    "identity_invalid_user_id_format",
		}).Warn("identity request failed: invalid user_id format")
		commonhttp.WriteErrorEnvelope(w, http.StatusBadRequest, commonhttp.CodeInvalidUserIDFormat, "invalid user_id format (must be UUID)", nil, "")
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
