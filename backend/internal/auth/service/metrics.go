package service

import (
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

func incrementRefreshTokensIssued() {
	metrics.RefreshTokensIssued.Inc()
}

func incrementRefreshTokensUsed() {
	metrics.RefreshTokensUsed.Inc()
}

func incrementRefreshTokensRevoked() {
	metrics.RefreshTokensRevoked.Inc()
}

func incrementRefreshTokensExpired() {
	metrics.RefreshTokensExpired.Inc()
}

func incrementAccessTokensRevoked() {
	metrics.AccessTokensRevoked.Inc()
}

func incrementAccessTokensIssued() {
	metrics.AccessTokensIssued.Inc()
}
