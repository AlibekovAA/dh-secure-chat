package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/httpmetrics"
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
	traceID := getTraceIDFromContext(ctx)

	if domainErr, ok := commonerrors.AsDomainError(err); ok {
		h.handleDomainError(w, r, domainErr)
		return
	}

	logFields := logger.Fields{
		"error":  err.Error(),
		"action": "unhandled_error",
	}
	if traceID != "" {
		logFields["trace_id"] = traceID
		w.Header().Set("X-Trace-ID", traceID)
	}

	h.log.WithFields(ctx, logFields).Errorf("unhandled error: %v", err)

	metrics.HTTPErrorsTotal.WithLabelValues(
		strconv.Itoa(http.StatusInternalServerError),
		httpmetrics.NormalizePath(r.URL.Path),
		r.Method,
	).Inc()

	WriteError(w, http.StatusInternalServerError, "internal server error")
}

func (h *ErrorHandler) handleDomainError(w http.ResponseWriter, r *http.Request, err commonerrors.DomainError) {
	ctx := r.Context()
	traceID := getTraceIDFromContext(ctx)

	var domainErr commonerrors.DomainError = err
	if traceID != "" && err.TraceID() == "" {
		domainErr = err.WithTraceID(traceID)
	}

	status := domainErr.HTTPStatus()
	message := domainErr.Message()

	logFields := logger.Fields{
		"error_code": domainErr.Code(),
		"category":   string(domainErr.Category()),
		"status":     status,
		"action":     "domain_error",
	}
	if traceID != "" {
		logFields["trace_id"] = traceID
	}

	if h.log.ShouldLog(logger.DEBUG) {
		h.log.WithFields(ctx, logFields).Debugf("domain error: %s", domainErr.Error())
	}

	metrics.DomainErrorsTotal.WithLabelValues(
		string(domainErr.Category()),
		domainErr.Code(),
		strconv.Itoa(status),
	).Inc()

	metrics.HTTPErrorsTotal.WithLabelValues(
		strconv.Itoa(status),
		httpmetrics.NormalizePath(r.URL.Path),
		r.Method,
	).Inc()

	if traceID != "" {
		w.Header().Set("X-Trace-ID", traceID)
	}

	WriteError(w, status, message)
}

func HandleError(w http.ResponseWriter, r *http.Request, err error, log *logger.Logger) {
	handler := NewErrorHandler(log)
	handler.HandleError(w, r, err)
}

func getTraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	val := ctx.Value(constants.TraceIDKey)
	if val == nil {
		return ""
	}
	traceID, ok := val.(string)
	if !ok || traceID == "" {
		return ""
	}
	return traceID
}
