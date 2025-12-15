package service

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrValidation         = errors.New("validation failed")
)
