package server

import (
	"net/http"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
)

type ServerConfig struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func DefaultServerConfig(port string) ServerConfig {
	return ServerConfig{
		Addr:              ":" + port,
		ReadHeaderTimeout: constants.ServerReadHeaderTimeout,
		ReadTimeout:       constants.ServerReadTimeout,
		WriteTimeout:      constants.ServerWriteTimeout,
		IdleTimeout:       constants.ServerIdleTimeout,
	}
}

func NewServer(cfg ServerConfig, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
}
