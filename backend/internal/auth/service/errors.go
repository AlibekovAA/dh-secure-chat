package service

import "errors"

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUsernameTaken        = errors.New("username already taken")
	ErrValidation           = errors.New("validation failed")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrRefreshTokenExpired  = errors.New("refresh token expired")
	ErrTooManyRefreshTokens = errors.New("too many active refresh tokens")
)
