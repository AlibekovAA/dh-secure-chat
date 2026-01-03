package jwtverify

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	commonhttp "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/http"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
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

func Middleware(secret string, log *logger.Logger, checker RevokedTokenChecker) func(next http.Handler) http.Handler {
	secretBytes := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, ok := ExtractTokenFromHeader(r)
			if !ok {
				metrics.JWTValidationsTotal.Inc()
				metrics.JWTValidationsFailed.Inc()
				log.Warnf("jwt auth failed path=%s: missing or invalid authorization header", r.URL.Path)
				commonhttp.WriteError(w, http.StatusUnauthorized, "missing or invalid authorization")
				return
			}

			metrics.JWTValidationsTotal.Inc()
			claims, err := parseToken(tokenString, secretBytes)
			if err != nil {
				metrics.JWTValidationsFailed.Inc()
				log.Warnf("jwt auth failed path=%s: %v", r.URL.Path, err)
				commonhttp.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			if checker != nil && claims.JTI != "" {
				revoked, err := checker.IsRevoked(r.Context(), claims.JTI)
				if err != nil {
					metrics.JWTValidationsFailed.Inc()
					log.Errorf("jwt auth failed path=%s: failed to check revoked token jti=%s: %v", r.URL.Path, claims.JTI, err)
					commonhttp.WriteError(w, http.StatusInternalServerError, "internal error")
					return
				}
				if revoked {
					metrics.JWTValidationsFailed.Inc()
					log.Warnf("jwt auth failed path=%s: token revoked jti=%s", r.URL.Path, claims.JTI)
					commonhttp.WriteError(w, http.StatusUnauthorized, "token revoked")
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
			commonhttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func ExtractTokenFromHeader(r *http.Request) (string, bool) {
	raw := r.Header.Get("Authorization")
	if raw == "" || !strings.HasPrefix(raw, "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(raw, "Bearer "), true
}

func ExtractAndParseToken(r *http.Request, secret []byte) (Claims, error) {
	tokenString, ok := ExtractTokenFromHeader(r)
	if !ok {
		return Claims{}, commonerrors.ErrInvalidToken
	}
	return parseToken(tokenString, secret)
}

func ParseToken(tokenString string, secret []byte) (Claims, error) {
	return parseToken(tokenString, secret)
}

func parseToken(tokenString string, secret []byte) (Claims, error) {
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, commonerrors.ErrInvalidTokenSigningMethod
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		if err == nil {
			err = commonerrors.ErrInvalidToken
		}
		if errors.Is(err, commonerrors.ErrInvalidToken) {
			return Claims{}, err
		}
		return Claims{}, commonerrors.ErrInvalidToken.WithCause(err)
	}

	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, commonerrors.ErrInvalidTokenClaims
	}

	sub, _ := mapClaims["sub"].(string)
	username, _ := mapClaims["usr"].(string)
	jti, _ := mapClaims["jti"].(string)
	if sub == "" || username == "" {
		return Claims{}, commonerrors.ErrMissingTokenClaims
	}

	return Claims{
		UserID:   sub,
		Username: username,
		JTI:      jti,
	}, nil
}
