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

	RefreshTokensRevoked = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "refresh_tokens_revoked_total",
			Help: "Total number of refresh tokens revoked",
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
)
