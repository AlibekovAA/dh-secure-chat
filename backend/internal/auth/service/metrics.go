package service

import (
	prommetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/prometheus"
)

func incrementRefreshTokensIssued() {
	prommetrics.RefreshTokensIssued.Inc()
}

func incrementRefreshTokensUsed() {
	prommetrics.RefreshTokensUsed.Inc()
}

func incrementRefreshTokensRevoked() {
	prommetrics.RefreshTokensRevoked.Inc()
}

func incrementRefreshTokensExpired() {
	prommetrics.RefreshTokensExpired.Inc()
}

func incrementAccessTokensRevoked() {
	prommetrics.AccessTokensRevoked.Inc()
}

func incrementAccessTokensIssued() {
	prommetrics.AccessTokensIssued.Inc()
}
