package constants

import "time"

const (
	UsernameMinLength  = 3
	UsernameMaxLength  = 32
	PasswordMinLength  = 8
	PasswordMaxLength  = 72
	JWTSecretMinLength = 32
	RefreshTokenSize   = 32

	MaxFileSizeBytes      = 50 * 1024 * 1024
	MaxVoiceSizeBytes     = 10 * 1024 * 1024
	MaxMessageLength      = 4000
	MaxSearchQueryLength  = 100
	MaxSearchResultsLimit = 100
	DefaultSearchLimit    = 20
	DefaultMaxRequestSize = 1 << 20
	MaxFileChunks         = 1000

	UserExistenceCacheTTL             = 5 * time.Minute
	UserExistenceCacheCleanupInterval = 1 * time.Minute

	RefreshTokenCacheTTL             = 1 * time.Minute
	RefreshTokenCacheCleanupInterval = 30 * time.Second

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

	DBPoolMaxOpenConns    = 50
	DBPoolMinOpenConns    = 10
	DBPoolConnMaxLifetime = 5 * time.Minute
	DBPoolConnMaxIdleTime = 10 * time.Minute
	DBPoolHealthCheck     = 1 * time.Minute
	DBPoolConnectTimeout  = 15 * time.Second
	DBPoolMaxAttempts     = 10
	DBPoolRetryDelay      = 1 * time.Second
	DBPoolMetricsInterval = 30 * time.Second
	DBQueryTimeout        = 30 * time.Second

	ServerReadHeaderTimeout = 10 * time.Second
	ServerReadTimeout       = 30 * time.Second
	ServerWriteTimeout      = 30 * time.Second
	ServerIdleTimeout       = 120 * time.Second

	ShutdownTimeout = 30 * time.Second
	DrainTimeout    = 10 * time.Second

	IdentityRequestTimeout = 5 * time.Second

	DefaultAuthHTTPPort = "8081"
	DefaultChatHTTPPort = "8082"

	DefaultCircuitBreakerThreshold = 500
	DefaultCircuitBreakerTimeout   = 15 * time.Second
	DefaultCircuitBreakerReset     = 10 * time.Second

	DefaultAuthRequestTimeout      = 30 * time.Second
	DefaultAccessTokenTTL          = 30 * time.Minute
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

	DefaultSearchUsersLimit = 20

	WebSocketReadBufferSize  = 1024
	WebSocketWriteBufferSize = 1024

	LoggerMaxSize    = 100
	LoggerMaxBackups = 3
	LoggerMaxAge     = 28
)

type TraceIDKeyType string

const TraceIDKey TraceIDKeyType = "trace_id"
