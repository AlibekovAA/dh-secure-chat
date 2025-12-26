package http

import (
	"net/http"
	"strconv"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

type ErrorHandler struct {
	log *logger.Logger
}

func NewErrorHandler(log *logger.Logger) *ErrorHandler {
	return &ErrorHandler{log: log}
}

func (h *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	ctx := r.Context()

	if domainErr, ok := commonerrors.AsDomainError(err); ok {
		h.handleDomainError(w, r, domainErr)
		return
	}

	h.log.WithFields(ctx, logger.Fields{
		"error":  err.Error(),
		"action": "unhandled_error",
	}).Errorf("unhandled error: %v", err)

	metrics.HTTPErrorsTotal.WithLabelValues(
		strconv.Itoa(http.StatusInternalServerError),
		r.URL.Path,
		r.Method,
	).Inc()

	WriteError(w, http.StatusInternalServerError, "internal server error")
}

func (h *ErrorHandler) handleDomainError(w http.ResponseWriter, r *http.Request, err commonerrors.DomainError) {
	status := err.HTTPStatus()
	message := err.Message()

	ctx := r.Context()
	h.log.WithFields(ctx, logger.Fields{
		"error_code": err.Code(),
		"category":   string(err.Category()),
		"status":     status,
		"action":     "domain_error",
	}).Debugf("domain error: %s", err.Error())

	metrics.DomainErrorsTotal.WithLabelValues(
		string(err.Category()),
		err.Code(),
		strconv.Itoa(status),
	).Inc()

	metrics.HTTPErrorsTotal.WithLabelValues(
		strconv.Itoa(status),
		r.URL.Path,
		r.Method,
	).Inc()

	WriteError(w, status, message)
}

func HandleError(w http.ResponseWriter, r *http.Request, err error, log *logger.Logger) {
	handler := NewErrorHandler(log)
	handler.HandleError(w, r, err)
}
