package http

import (
	"net/http"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/httpmetrics"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func BuildBaseHandler(appName string, log *logger.Logger, handler http.Handler) http.Handler {
	metrics := httpmetrics.New(appName)
	recovery := RecoveryMiddleware(log)
	traceID := TraceIDMiddleware
	maxRequestSize := MaxRequestSizeMiddleware(constants.DefaultMaxRequestSize)
	securityHeaders := SecurityHeadersMiddleware
	csp := ContentSecurityPolicyMiddleware("")

	return securityHeaders(csp(recovery(traceID(maxRequestSize(metrics.Wrap(handler))))))
}
