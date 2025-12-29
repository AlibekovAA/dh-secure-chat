package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RefreshTokensIssued = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_issued_total",
			Help: "Total number of refresh tokens issued",
		},
	)

	RefreshTokensUsed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_used_total",
			Help: "Total number of refresh tokens used",
		},
	)

	RefreshTokensRevoked = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_revoked_total",
			Help: "Total number of refresh tokens revoked",
		},
	)

	RefreshTokensExpired = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_expired_total",
			Help: "Total number of expired refresh tokens",
		},
	)

	RefreshTokensCleanupDeleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_cleanup_deleted_total",
			Help: "Total number of expired refresh tokens deleted during cleanup",
		},
	)

	AccessTokensIssued = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "access_tokens_issued_total",
			Help: "Total number of access tokens issued",
		},
	)

	AccessTokensRevoked = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "access_tokens_revoked_total",
			Help: "Total number of access tokens revoked",
		},
	)

	RevokedTokensCleanupDeleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "revoked_tokens_cleanup_deleted_total",
			Help: "Total number of expired revoked tokens deleted during cleanup",
		},
	)

	JWTValidationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jwt_validations_total",
			Help: "Total number of JWT validations",
		},
	)

	JWTValidationsFailed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jwt_validations_failed_total",
			Help: "Total number of failed JWT validations",
		},
	)

	JWTRevokedChecksTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jwt_revoked_checks_total",
			Help: "Total number of revoked token checks",
		},
	)
)
