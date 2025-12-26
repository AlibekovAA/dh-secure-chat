package service

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	commoncrypto "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/crypto"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	userdomain "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/domain"
)

type TokenIssuer struct {
	jwtSecret      []byte
	idGenerator    commoncrypto.IDGenerator
	clock          clock.Clock
	accessTokenTTL time.Duration
}

func NewTokenIssuer(
	jwtSecret string,
	idGenerator commoncrypto.IDGenerator,
	accessTokenTTL time.Duration,
	clock clock.Clock,
) *TokenIssuer {
	return &TokenIssuer{
		jwtSecret:      []byte(jwtSecret),
		idGenerator:    idGenerator,
		clock:          clock,
		accessTokenTTL: accessTokenTTL,
	}
}

func (ti *TokenIssuer) IssueAccessToken(user userdomain.User) (string, string, error) {
	jti, err := ti.idGenerator.NewID()
	if err != nil {
		return "", "", err
	}

	now := ti.clock.Now()
	expiresAt := now.Add(ti.accessTokenTTL)
	claims := jwt.MapClaims{
		"sub": string(user.ID),
		"usr": user.Username,
		"jti": jti,
		"exp": expiresAt.Unix(),
		"iat": now.Unix(),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := t.SignedString(ti.jwtSecret)
	if err != nil {
		return "", "", err
	}

	incrementAccessTokensIssued()
	return tokenString, jti, nil
}

func (ti *TokenIssuer) ParseToken(tokenString string) (jwtverify.Claims, error) {
	return jwtverify.ParseToken(tokenString, ti.jwtSecret)
}
