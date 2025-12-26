package http

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanup  *time.Ticker
}

func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
		cleanup:  time.NewTicker(5 * time.Minute),
	}

	go rl.cleanupLimiters()

	return rl
}

func (rl *RateLimiter) cleanupLimiters() {
	for range rl.cleanup.C {
		rl.mu.Lock()
		for key, limiter := range rl.limiters {
			if limiter.Allow() {
				delete(rl.limiters, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		limiter, exists = rl.limiters[key]
		if !exists {
			limiter = rate.NewLimiter(rl.rate, rl.burst)
			rl.limiters[key] = limiter
		}
		rl.mu.Unlock()
	}

	return limiter
}

func (rl *RateLimiter) Allow(key string) bool {
	return rl.getLimiter(key).Allow()
}

func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := getClientKey(r)

			if !rl.Allow(key) {
				metrics.RateLimitBlocked.WithLabelValues(r.URL.Path, "general").Inc()
				WriteError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getClientKey(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}

	userID := ""
	if claims, ok := r.Context().Value("jwt_claims").(struct {
		UserID   string
		Username string
		JTI      string
	}); ok {
		userID = claims.UserID
	}

	if userID != "" {
		return userID
	}

	return ip
}

type StrictRateLimiter struct {
	loginLimiter    *RateLimiter
	registerLimiter *RateLimiter
	generalLimiter  *RateLimiter
}

func NewStrictRateLimiter() *StrictRateLimiter {
	return &StrictRateLimiter{
		loginLimiter:    NewRateLimiter(5.0, 5),
		registerLimiter: NewRateLimiter(2.0, 2),
		generalLimiter:  NewRateLimiter(100.0, 100),
	}
}

func (srl *StrictRateLimiter) MiddlewareForPath(path string) func(http.Handler) http.Handler {
	var limiter *RateLimiter
	var limiterType string

	switch path {
	case "/api/auth/login":
		limiter = srl.loginLimiter
		limiterType = "login"
	case "/api/auth/register":
		limiter = srl.registerLimiter
		limiterType = "register"
	default:
		limiter = srl.generalLimiter
		limiterType = "general"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := getClientKey(r)

			if !limiter.Allow(key) {
				metrics.RateLimitBlocked.WithLabelValues(path, limiterType).Inc()
				WriteError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
