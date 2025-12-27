package config

import (
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"

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
		CircuitBreakerThreshold: int32(getIntEnv(prefix+"_CIRCUIT_BREAKER_THRESHOLD", 5)),
		CircuitBreakerTimeout:   getDurationEnv(prefix+"_CIRCUIT_BREAKER_TIMEOUT", 5*time.Second),
		CircuitBreakerReset:     getDurationEnv(prefix+"_CIRCUIT_BREAKER_RESET", 30*time.Second),
	}, nil
}

func LoadAuthConfig() (AuthConfig, error) {
	base, err := loadBaseConfig("AUTH", "8081")
	if err != nil {
		return AuthConfig{}, err
	}

	cfg := AuthConfig{
		BaseConfig:              base,
		RequestTimeout:          getDurationEnv("AUTH_REQUEST_TIMEOUT", 5*time.Second),
		AccessTokenTTL:          getDurationEnv("AUTH_ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:         getDurationEnv("AUTH_REFRESH_TOKEN_TTL", 7*24*time.Hour),
		MaxRefreshTokensPerUser: getIntEnv("AUTH_MAX_REFRESH_TOKENS_PER_USER", 5),
	}

	if err := validate.Struct(cfg); err != nil {
		return AuthConfig{}, commonerrors.ErrInternalError.WithCause(err)
	}

	return cfg, nil
}

func LoadChatConfig() (ChatConfig, error) {
	base, err := loadBaseConfig("CHAT", "8082")
	if err != nil {
		return ChatConfig{}, err
	}

	cfg := ChatConfig{
		BaseConfig:              base,
		WebSocketWriteWait:      getDurationEnv("CHAT_WS_WRITE_WAIT", 10*time.Second),
		WebSocketPongWait:       getDurationEnv("CHAT_WS_PONG_WAIT", 60*time.Second),
		WebSocketPingPeriod:     getDurationEnv("CHAT_WS_PING_PERIOD", 54*time.Second),
		WebSocketMaxMsgSize:     getInt64Env("CHAT_WS_MAX_MSG_SIZE", 20*1024*1024),
		WebSocketSendBufSize:    getIntEnv("CHAT_WS_SEND_BUF_SIZE", 256),
		WebSocketAuthTimeout:    getDurationEnv("CHAT_WS_AUTH_TIMEOUT", 10*time.Second),
		WebSocketSendTimeout:    getDurationEnv("CHAT_WS_SEND_TIMEOUT", 2*time.Second),
		LastSeenUpdateInterval:  getDurationEnv("CHAT_LAST_SEEN_INTERVAL", 1*time.Minute),
		RequestTimeout:          getDurationEnv("CHAT_REQUEST_TIMEOUT", 5*time.Second),
		SearchTimeout:           getDurationEnv("CHAT_SEARCH_TIMEOUT", 10*time.Second),
		WebSocketMaxConnections: getIntEnv("CHAT_WS_MAX_CONNECTIONS", 10000),
	}

	if err := validate.Struct(cfg); err != nil {
		return ChatConfig{}, commonerrors.ErrInternalError.WithCause(err)
	}

	return cfg, nil
}

func validateJWTSecret(secret string) error {
	if len(secret) < 32 {
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
