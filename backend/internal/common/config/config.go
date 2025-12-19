package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

var (
	ErrMissingRequiredEnv = errors.New("missing required environment variable")
	ErrInvalidJWTSecret   = errors.New("JWT_SECRET must be at least 32 bytes")
)

type AuthConfig struct {
	HTTPPort       string
	DatabaseURL    string
	JWTSecret      string
	RequestTimeout time.Duration
}

type ChatConfig struct {
	HTTPPort               string
	DatabaseURL            string
	JWTSecret              string
	WebSocketWriteWait     time.Duration
	WebSocketPongWait      time.Duration
	WebSocketPingPeriod    time.Duration
	WebSocketMaxMsgSize    int64
	WebSocketSendBufSize   int
	WebSocketAuthTimeout   time.Duration
	LastSeenUpdateInterval time.Duration
	RequestTimeout         time.Duration
	SearchTimeout          time.Duration
}

func LoadAuthConfig() (AuthConfig, error) {
	jwtSecret, err := mustEnv("JWT_SECRET")
	if err != nil {
		return AuthConfig{}, err
	}

	if err := validateJWTSecret(jwtSecret); err != nil {
		return AuthConfig{}, err
	}

	databaseURL, err := mustEnv("DATABASE_URL")
	if err != nil {
		return AuthConfig{}, err
	}

	return AuthConfig{
		HTTPPort:       getEnv("AUTH_HTTP_PORT", "8081"),
		DatabaseURL:    databaseURL,
		JWTSecret:      jwtSecret,
		RequestTimeout: getDurationEnv("AUTH_REQUEST_TIMEOUT", 5*time.Second),
	}, nil
}

func LoadChatConfig() (ChatConfig, error) {
	jwtSecret, err := mustEnv("JWT_SECRET")
	if err != nil {
		return ChatConfig{}, err
	}

	if err := validateJWTSecret(jwtSecret); err != nil {
		return ChatConfig{}, err
	}

	databaseURL, err := mustEnv("DATABASE_URL")
	if err != nil {
		return ChatConfig{}, err
	}

	return ChatConfig{
		HTTPPort:               getEnv("CHAT_HTTP_PORT", "8082"),
		DatabaseURL:            databaseURL,
		JWTSecret:              jwtSecret,
		WebSocketWriteWait:     getDurationEnv("CHAT_WS_WRITE_WAIT", 10*time.Second),
		WebSocketPongWait:      getDurationEnv("CHAT_WS_PONG_WAIT", 60*time.Second),
		WebSocketPingPeriod:    getDurationEnv("CHAT_WS_PING_PERIOD", 54*time.Second),
		WebSocketMaxMsgSize:    getInt64Env("CHAT_WS_MAX_MSG_SIZE", 20*1024*1024),
		WebSocketSendBufSize:   getIntEnv("CHAT_WS_SEND_BUF_SIZE", 256),
		WebSocketAuthTimeout:   getDurationEnv("CHAT_WS_AUTH_TIMEOUT", 10*time.Second),
		LastSeenUpdateInterval: getDurationEnv("CHAT_LAST_SEEN_INTERVAL", 1*time.Minute),
		RequestTimeout:         getDurationEnv("CHAT_REQUEST_TIMEOUT", 5*time.Second),
		SearchTimeout:          getDurationEnv("CHAT_SEARCH_TIMEOUT", 10*time.Second),
	}, nil
}

func validateJWTSecret(secret string) error {
	if len(secret) < 32 {
		return fmt.Errorf("%w: got %d bytes", ErrInvalidJWTSecret, len(secret))
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
		return "", fmt.Errorf("%w: %s", ErrMissingRequiredEnv, key)
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
