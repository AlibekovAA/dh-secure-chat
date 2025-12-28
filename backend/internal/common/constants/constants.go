package constants

import "time"

const (
	UsernameMinLength  = 3
	UsernameMaxLength  = 32
	PasswordMinLength  = 8
	PasswordMaxLength  = 72
	JWTSecretMinLength = 32

	MaxFileSizeBytes      = 50 * 1024 * 1024
	MaxVoiceSizeBytes     = 10 * 1024 * 1024
	MaxMessageLength      = 4000
	MaxSearchQueryLength  = 100
	MaxSearchResultsLimit = 100
	DefaultSearchLimit    = 20
	DefaultMaxRequestSize = 1 << 20

	UserExistenceCacheTTL             = 5 * time.Minute
	UserExistenceCacheCleanupInterval = 1 * time.Minute

	FileTransferTimeout = 10 * time.Minute
	IdempotencyTTL      = 5 * time.Minute

	WebSocketProcessorWorkers            = 10
	WebSocketProcessorQueueSize          = 10000
	WebSocketProcessorDefaultQueueSize   = 1000
	WebSocketProcessorTimeout            = 30 * time.Second
	WebSocketDebugSampleRate             = 0.01
	WebSocketShutdownNotificationTimeout = 5 * time.Second
	WebSocketFileTrackerCleanupInterval  = 1 * time.Minute

	LastSeenQueueSize     = 100
	LastSeenBatchSize     = 100
	LastSeenFlushEvery    = 500 * time.Millisecond
	LastSeenUpdateTimeout = 3 * time.Second

	RateLimitCleanupInterval = 5 * time.Minute

	DBPoolMaxOpenConns    = 25
	DBPoolMinOpenConns    = 5
	DBPoolConnMaxLifetime = 5 * time.Minute
	DBPoolConnMaxIdleTime = 10 * time.Minute
	DBPoolHealthCheck     = 1 * time.Minute
	DBPoolConnectTimeout  = 5 * time.Second
	DBPoolMaxAttempts     = 10
	DBPoolRetryDelay      = 1 * time.Second
	DBPoolMetricsInterval = 30 * time.Second

	ServerReadHeaderTimeout = 5 * time.Second
	ServerReadTimeout       = 10 * time.Second
	ServerWriteTimeout      = 10 * time.Second
	ServerIdleTimeout       = 120 * time.Second

	ShutdownTimeout = 30 * time.Second
	DrainTimeout    = 10 * time.Second

	IdentityRequestTimeout = 5 * time.Second

	DefaultAuthHTTPPort = "8081"
	DefaultChatHTTPPort = "8082"

	DefaultCircuitBreakerThreshold = 5
	DefaultCircuitBreakerTimeout   = 5 * time.Second
	DefaultCircuitBreakerReset     = 30 * time.Second

	DefaultAuthRequestTimeout      = 5 * time.Second
	DefaultAccessTokenTTL          = 15 * time.Minute
	DefaultRefreshTokenTTL         = 7 * 24 * time.Hour
	DefaultMaxRefreshTokensPerUser = 5

	DefaultWebSocketWriteWait      = 10 * time.Second
	DefaultWebSocketPongWait       = 60 * time.Second
	DefaultWebSocketPingPeriod     = 54 * time.Second
	DefaultWebSocketMaxMsgSize     = 20 * 1024 * 1024
	DefaultWebSocketSendBufSize    = 256
	DefaultWebSocketAuthTimeout    = 10 * time.Second
	DefaultWebSocketSendTimeout    = 2 * time.Second
	DefaultLastSeenUpdateInterval  = 1 * time.Minute
	DefaultChatRequestTimeout      = 5 * time.Second
	DefaultSearchTimeout           = 10 * time.Second
	DefaultWebSocketMaxConnections = 10000

	RateLimitLoginRequestsPerSecond    = 3.0
	RateLimitLoginBurst                = 3
	RateLimitRegisterRequestsPerSecond = 2.0
	RateLimitRegisterBurst             = 1
	RateLimitRefreshRequestsPerSecond  = 1.0
	RateLimitRefreshBurst              = 3
	RateLimitLogoutRequestsPerSecond   = 1.0
	RateLimitLogoutBurst               = 2
	RateLimitRevokeRequestsPerSecond   = 1.0
	RateLimitRevokeBurst               = 2
	RateLimitGeneralRequestsPerSecond  = 100.0
	RateLimitGeneralBurst              = 100

	DefaultSearchUsersLimit = 20

	WebSocketReadBufferSize  = 1024
	WebSocketWriteBufferSize = 1024

	LoggerMaxSize    = 100
	LoggerMaxBackups = 3
	LoggerMaxAge     = 28
)
