package jwtverify

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

var (
	jwtValidationsTotal   = expvar.NewInt("jwt_validations_total")
	jwtValidationsFailed  = expvar.NewInt("jwt_validations_failed")
	jwtRevokedChecksTotal = expvar.NewInt("jwt_revoked_checks_total")
)

type RevokedTokenChecker interface {
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

type Claims struct {
	UserID   string
	Username string
	JTI      string
}

type contextKey string

const claimsKey contextKey = "jwt_claims"

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func Middleware(secret string, log *logger.Logger, checker RevokedTokenChecker) func(next http.Handler) http.Handler {
	secretBytes := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("Authorization")
			if raw == "" || !strings.HasPrefix(raw, "Bearer ") {
				jwtValidationsTotal.Add(1)
				jwtValidationsFailed.Add(1)
				log.Warnf("jwt auth failed path=%s: missing or invalid authorization header", r.URL.Path)
				writeError(w, http.StatusUnauthorized, "missing or invalid authorization")
				return
			}

			tokenString := strings.TrimPrefix(raw, "Bearer ")
			jwtValidationsTotal.Add(1)
			claims, err := parseToken(tokenString, secretBytes)
			if err != nil {
				jwtValidationsFailed.Add(1)
				log.Warnf("jwt auth failed path=%s: %v", r.URL.Path, err)
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			if checker != nil && claims.JTI != "" {
				jwtRevokedChecksTotal.Add(1)
				revoked, err := checker.IsRevoked(r.Context(), claims.JTI)
				if err != nil {
					jwtValidationsFailed.Add(1)
					log.Errorf("jwt auth failed path=%s: failed to check revoked token jti=%s: %v", r.URL.Path, claims.JTI, err)
					writeError(w, http.StatusInternalServerError, "internal error")
					return
				}
				if revoked {
					jwtValidationsFailed.Add(1)
					log.Warnf("jwt auth failed path=%s: token revoked jti=%s", r.URL.Path, claims.JTI)
					writeError(w, http.StatusUnauthorized, "token revoked")
					return
				}
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func FromContext(ctx context.Context) (Claims, bool) {
	val := ctx.Value(claimsKey)
	claims, ok := val.(Claims)
	return claims, ok
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func ParseToken(tokenString string, secret []byte) (Claims, error) {
	return parseToken(tokenString, secret)
}

func parseToken(tokenString string, secret []byte) (Claims, error) {
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method)
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		if err == nil {
			err = commonerrors.ErrInvalidToken
		}
		return Claims{}, fmt.Errorf("failed to parse token: %w", err)
	}

	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, fmt.Errorf("invalid claims type: expected MapClaims, got %T", parsed.Claims)
	}

	sub, _ := mapClaims["sub"].(string)
	username, _ := mapClaims["usr"].(string)
	jti, _ := mapClaims["jti"].(string)
	if sub == "" || username == "" {
		return Claims{}, fmt.Errorf("missing required claims: sub=%q, usr=%q", sub, username)
	}

	return Claims{
		UserID:   sub,
		Username: username,
		JTI:      jti,
	}, nil
}
