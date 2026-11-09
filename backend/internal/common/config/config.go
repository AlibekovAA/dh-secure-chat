package config

import (
	"fmt"
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
	SecureCookies           bool
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

	httpPort := getEnv(prefix+"_HTTP_PORT", defaultPort)
	port, err := strconv.Atoi(httpPort)
	if err != nil || port < 1 || port > 65535 {
		return BaseConfig{}, fmt.Errorf("invalid %s_HTTP_PORT: must be between 1 and 65535", prefix)
	}
	threshold, err := getIntEnv(prefix+"_CIRCUIT_BREAKER_THRESHOLD", constants.DefaultCircuitBreakerThreshold)
	if err != nil {
		return BaseConfig{}, err
	}
	timeout, err := getDurationEnv(prefix+"_CIRCUIT_BREAKER_TIMEOUT", constants.DefaultCircuitBreakerTimeout)
	if err != nil {
		return BaseConfig{}, err
	}
	reset, err := getDurationEnv(prefix+"_CIRCUIT_BREAKER_RESET", constants.DefaultCircuitBreakerReset)
	if err != nil {
		return BaseConfig{}, err
	}

	return BaseConfig{
		DatabaseURL:             databaseURL,
		JWTSecret:               jwtSecret,
		HTTPPort:                httpPort,
		CircuitBreakerThreshold: int32(threshold),
		CircuitBreakerTimeout:   timeout,
		CircuitBreakerReset:     reset,
	}, nil
}

func LoadAuthConfig() (AuthConfig, error) {
	base, err := loadBaseConfig("AUTH", constants.DefaultAuthHTTPPort)
	if err != nil {
		return AuthConfig{}, err
	}

	requestTimeout, err := getDurationEnv("AUTH_REQUEST_TIMEOUT", constants.DefaultAuthRequestTimeout)
	if err != nil {
		return AuthConfig{}, err
	}
	accessTokenTTL, err := getDurationEnv("AUTH_ACCESS_TOKEN_TTL", constants.DefaultAccessTokenTTL)
	if err != nil {
		return AuthConfig{}, err
	}
	refreshTokenTTL, err := getDurationEnv("AUTH_REFRESH_TOKEN_TTL", constants.DefaultRefreshTokenTTL)
	if err != nil {
		return AuthConfig{}, err
	}
	maxRefreshTokens, err := getIntEnv("AUTH_MAX_REFRESH_TOKENS_PER_USER", constants.DefaultMaxRefreshTokensPerUser)
	if err != nil {
		return AuthConfig{}, err
	}
	secureCookies, err := getBoolEnv("AUTH_SECURE_COOKIES", true)
	if err != nil {
		return AuthConfig{}, err
	}

	cfg := AuthConfig{
		BaseConfig:              base,
		RequestTimeout:          requestTimeout,
		AccessTokenTTL:          accessTokenTTL,
		RefreshTokenTTL:         refreshTokenTTL,
		MaxRefreshTokensPerUser: maxRefreshTokens,
		SecureCookies:           secureCookies,
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

	writeWait, err := getDurationEnv("CHAT_WS_WRITE_WAIT", constants.DefaultWebSocketWriteWait)
	if err != nil {
		return ChatConfig{}, err
	}
	pongWait, err := getDurationEnv("CHAT_WS_PONG_WAIT", constants.DefaultWebSocketPongWait)
	if err != nil {
		return ChatConfig{}, err
	}
	pingPeriod, err := getDurationEnv("CHAT_WS_PING_PERIOD", constants.DefaultWebSocketPingPeriod)
	if err != nil {
		return ChatConfig{}, err
	}
	maxMessageSize, err := getInt64Env("CHAT_WS_MAX_MSG_SIZE", constants.DefaultWebSocketMaxMsgSize)
	if err != nil {
		return ChatConfig{}, err
	}
	sendBufferSize, err := getIntEnv("CHAT_WS_SEND_BUF_SIZE", constants.DefaultWebSocketSendBufSize)
	if err != nil {
		return ChatConfig{}, err
	}
	authTimeout, err := getDurationEnv("CHAT_WS_AUTH_TIMEOUT", constants.DefaultWebSocketAuthTimeout)
	if err != nil {
		return ChatConfig{}, err
	}
	sendTimeout, err := getDurationEnv("CHAT_WS_SEND_TIMEOUT", constants.DefaultWebSocketSendTimeout)
	if err != nil {
		return ChatConfig{}, err
	}
	requestTimeout, err := getDurationEnv("CHAT_REQUEST_TIMEOUT", constants.DefaultChatRequestTimeout)
	if err != nil {
		return ChatConfig{}, err
	}
	searchTimeout, err := getDurationEnv("CHAT_SEARCH_TIMEOUT", constants.DefaultSearchTimeout)
	if err != nil {
		return ChatConfig{}, err
	}
	maxConnections, err := getIntEnv("CHAT_WS_MAX_CONNECTIONS", constants.DefaultWebSocketMaxConnections)
	if err != nil {
		return ChatConfig{}, err
	}

	cfg := ChatConfig{
		BaseConfig:              base,
		WebSocketWriteWait:      writeWait,
		WebSocketPongWait:       pongWait,
		WebSocketPingPeriod:     pingPeriod,
		WebSocketMaxMsgSize:     maxMessageSize,
		WebSocketSendBufSize:    sendBufferSize,
		WebSocketAuthTimeout:    authTimeout,
		WebSocketSendTimeout:    sendTimeout,
		RequestTimeout:          requestTimeout,
		SearchTimeout:           searchTimeout,
		WebSocketMaxConnections: maxConnections,
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

func getDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return d, nil
}

func getIntEnv(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return i, nil
}

func getInt64Env(key string, fallback int64) (int64, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return i, nil
}

func getBoolEnv(key string, fallback bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}
