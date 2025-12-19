package service

import "expvar"

var (
	refreshTokensIssued  = expvar.NewInt("refresh_tokens_issued")
	refreshTokensUsed    = expvar.NewInt("refresh_tokens_used")
	refreshTokensRevoked = expvar.NewInt("refresh_tokens_revoked")
	refreshTokensExpired = expvar.NewInt("refresh_tokens_expired")
	accessTokensRevoked  = expvar.NewInt("access_tokens_revoked")
	accessTokensIssued   = expvar.NewInt("access_tokens_issued")
)
