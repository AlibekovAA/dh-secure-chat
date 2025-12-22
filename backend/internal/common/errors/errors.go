package commonerrors

import "errors"

var (
	ErrMissingRequiredEnv    = errors.New("missing required environment variable")
	ErrInvalidJWTSecret      = errors.New("JWT_SECRET must be at least 32 bytes")
	ErrUserNotConnected      = errors.New("user not connected")
	ErrEmptyQuery            = errors.New("query is empty")
	ErrIdentityKeyNotFound   = errors.New("identity key not found")
	ErrInvalidPublicKey      = errors.New("invalid public key")
	ErrTransferNotFound      = errors.New("transfer not found")
	ErrTransferAlreadyExists = errors.New("transfer already exists")
	ErrInvalidChunkIndex     = errors.New("invalid chunk index")
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrCircuitOpen           = errors.New("circuit breaker is open")
	ErrInvalidToken          = errors.New("token is not valid")
	ErrEmptyUUID             = errors.New("uuid cannot be empty")
)
