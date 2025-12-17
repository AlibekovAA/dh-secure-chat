package jwtverify

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type Claims struct {
	UserID   string
	Username string
}

type contextKey string

const claimsKey contextKey = "jwt_claims"

func Middleware(secret string, log *logger.Logger) func(next http.Handler) http.Handler {
	secretBytes := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("Authorization")
			if raw == "" || !strings.HasPrefix(raw, "Bearer ") {
				log.Warnf("jwt auth failed path=%s: missing or invalid authorization header", r.URL.Path)
				commonhttp.WriteError(w, http.StatusUnauthorized, "missing or invalid authorization")
				return
			}

			tokenString := strings.TrimPrefix(raw, "Bearer ")
			claims, err := parseToken(tokenString, secretBytes)
			if err != nil {
				log.Warnf("jwt auth failed path=%s: %v", r.URL.Path, err)
				commonhttp.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
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

func ParseToken(tokenString string, secret []byte) (Claims, error) {
	return parseToken(tokenString, secret)
}

func parseToken(tokenString string, secret []byte) (Claims, error) {
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		if err == nil {
			err = errors.New("token is not valid")
		}
		return Claims{}, err
	}

	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, errors.New("invalid claims type")
	}

	sub, _ := mapClaims["sub"].(string)
	username, _ := mapClaims["usr"].(string)
	if sub == "" || username == "" {
		return Claims{}, errors.New("missing sub or usr claims")
	}

	return Claims{
		UserID:   sub,
		Username: username,
	}, nil
}
