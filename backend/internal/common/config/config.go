package config

import (
	"fmt"
	"os"
)

type AuthConfig struct {
	HTTPPort    string
	DatabaseURL string
	JWTSecret   string
}

type ChatConfig struct {
	HTTPPort    string
	DatabaseURL string
	JWTSecret   string
}

func LoadAuthConfig() AuthConfig {
	jwtSecret := mustEnv("JWT_SECRET")
	validateJWTSecret(jwtSecret)

	return AuthConfig{
		HTTPPort:    getEnv("AUTH_HTTP_PORT", "8081"),
		DatabaseURL: mustEnv("DATABASE_URL"),
		JWTSecret:   jwtSecret,
	}
}

func LoadChatConfig() ChatConfig {
	jwtSecret := mustEnv("JWT_SECRET")
	validateJWTSecret(jwtSecret)

	return ChatConfig{
		HTTPPort:    getEnv("CHAT_HTTP_PORT", "8082"),
		DatabaseURL: mustEnv("DATABASE_URL"),
		JWTSecret:   jwtSecret,
	}
}

func validateJWTSecret(secret string) {
	if len(secret) < 32 {
		panic(fmt.Sprintf("JWT_SECRET must be at least 32 bytes, got %d bytes", len(secret)))
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		panic("missing required env: " + key)
	}
	return v
}
