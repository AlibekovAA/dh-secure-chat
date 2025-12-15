package config

import (
	"os"
)

type AuthConfig struct {
	HTTPPort    string
	DatabaseURL string
	JWTSecret   string
}

func LoadAuthConfig() AuthConfig {
	return AuthConfig{
		HTTPPort:    getEnv("AUTH_HTTP_PORT", "8081"),
		DatabaseURL: mustEnv("DATABASE_URL"),
		JWTSecret:   mustEnv("JWT_SECRET"),
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
