package config

import (
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

type BaseConfig struct {
	DatabaseURL             string        `validate:"required,url"`
	JWTSecret               string        `validate:"required"`
	HTTPPort                string        `validate:"required"`
	CircuitBreakerThreshold int32         `validate:"gt=0"`
	CircuitBreakerTimeout   time.Duration `validate:"gt=0"`
	CircuitBreakerReset     time.Duration `validate:"gt=0"`
}

type AuthConfig struct {
	BaseConfig
	RequestTimeout          time.Duration `validate:"gt=0"`
	AccessTokenTTL          time.Duration `validate:"gt=0"`
	RefreshTokenTTL         time.Duration `validate:"gt=0"`
	MaxRefreshTokensPerUser int           `validate:"gt=0"`
}

type ChatConfig struct {
	BaseConfig
	WebSocketWriteWait      time.Duration `validate:"gt=0"`
	WebSocketPongWait       time.Duration `validate:"gt=0"`
	WebSocketPingPeriod     time.Duration `validate:"gt=0"`
	WebSocketMaxMsgSize     int64         `validate:"gt=0"`
	WebSocketSendBufSize    int           `validate:"gt=0"`
	WebSocketAuthTimeout    time.Duration `validate:"gt=0"`
	WebSocketSendTimeout    time.Duration `validate:"gt=0"`
	LastSeenUpdateInterval  time.Duration `validate:"gt=0"`
	RequestTimeout          time.Duration `validate:"gt=0"`
	SearchTimeout           time.Duration `validate:"gt=0"`
	WebSocketMaxConnections int           `validate:"gt=0"`
}

var validate = validator.New()

func loadBaseConfig(prefix string, defaultPort string) (BaseConfig, error) {
	jwtSecret, err := mustEnv("JWT_SECRET")
	if err != nil {
		return BaseConfig{}, err
	}

	if err := validateJWTSecret(jwtSecret); err != nil {
		return BaseConfig{}, err
	}

	databaseURL, err := mustEnv("DATABASE_URL")
	if err != nil {
		return BaseConfig{}, err
	}

	return BaseConfig{
		DatabaseURL:             databaseURL,
		JWTSecret:               jwtSecret,
		HTTPPort:                getEnv(prefix+"_HTTP_PORT", defaultPort),
		CircuitBreakerThreshold: int32(getIntEnv(prefix+"_CIRCUIT_BREAKER_THRESHOLD", constants.DefaultCircuitBreakerThreshold)),
		CircuitBreakerTimeout:   getDurationEnv(prefix+"_CIRCUIT_BREAKER_TIMEOUT", constants.DefaultCircuitBreakerTimeout),
		CircuitBreakerReset:     getDurationEnv(prefix+"_CIRCUIT_BREAKER_RESET", constants.DefaultCircuitBreakerReset),
	}, nil
}

func LoadAuthConfig() (AuthConfig, error) {
	base, err := loadBaseConfig("AUTH", constants.DefaultAuthHTTPPort)
	if err != nil {
		return AuthConfig{}, err
	}

	cfg := AuthConfig{
		BaseConfig:              base,
		RequestTimeout:          getDurationEnv("AUTH_REQUEST_TIMEOUT", constants.DefaultAuthRequestTimeout),
		AccessTokenTTL:          getDurationEnv("AUTH_ACCESS_TOKEN_TTL", constants.DefaultAccessTokenTTL),
		RefreshTokenTTL:         getDurationEnv("AUTH_REFRESH_TOKEN_TTL", constants.DefaultRefreshTokenTTL),
		MaxRefreshTokensPerUser: getIntEnv("AUTH_MAX_REFRESH_TOKENS_PER_USER", constants.DefaultMaxRefreshTokensPerUser),
	}

	if err := validate.Struct(cfg); err != nil {
		return AuthConfig{}, commonerrors.ErrInternalError.WithCause(err)
	}

	return cfg, nil
}

func LoadChatConfig() (ChatConfig, error) {
	base, err := loadBaseConfig("CHAT", constants.DefaultChatHTTPPort)
	if err != nil {
		return ChatConfig{}, err
	}

	cfg := ChatConfig{
		BaseConfig:              base,
		WebSocketWriteWait:      getDurationEnv("CHAT_WS_WRITE_WAIT", constants.DefaultWebSocketWriteWait),
		WebSocketPongWait:       getDurationEnv("CHAT_WS_PONG_WAIT", constants.DefaultWebSocketPongWait),
		WebSocketPingPeriod:     getDurationEnv("CHAT_WS_PING_PERIOD", constants.DefaultWebSocketPingPeriod),
		WebSocketMaxMsgSize:     getInt64Env("CHAT_WS_MAX_MSG_SIZE", constants.DefaultWebSocketMaxMsgSize),
		WebSocketSendBufSize:    getIntEnv("CHAT_WS_SEND_BUF_SIZE", constants.DefaultWebSocketSendBufSize),
		WebSocketAuthTimeout:    getDurationEnv("CHAT_WS_AUTH_TIMEOUT", constants.DefaultWebSocketAuthTimeout),
		WebSocketSendTimeout:    getDurationEnv("CHAT_WS_SEND_TIMEOUT", constants.DefaultWebSocketSendTimeout),
		LastSeenUpdateInterval:  getDurationEnv("CHAT_LAST_SEEN_INTERVAL", constants.DefaultLastSeenUpdateInterval),
		RequestTimeout:          getDurationEnv("CHAT_REQUEST_TIMEOUT", constants.DefaultChatRequestTimeout),
		SearchTimeout:           getDurationEnv("CHAT_SEARCH_TIMEOUT", constants.DefaultSearchTimeout),
		WebSocketMaxConnections: getIntEnv("CHAT_WS_MAX_CONNECTIONS", constants.DefaultWebSocketMaxConnections),
	}

	if err := validate.Struct(cfg); err != nil {
		return ChatConfig{}, commonerrors.ErrInternalError.WithCause(err)
	}

	return cfg, nil
}

func validateJWTSecret(secret string) error {
	if len(secret) < constants.JWTSecretMinLength {
		return commonerrors.ErrInvalidJWTSecret
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func mustEnv(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", commonerrors.ErrMissingRequiredEnv
	}
	return v, nil
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getIntEnv(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getInt64Env(key string, fallback int64) int64 {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return i
}
