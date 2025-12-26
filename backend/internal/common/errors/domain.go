package commonerrors

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorCategory string

const (
	CategoryValidation   ErrorCategory = "VALIDATION"
	CategoryAuth         ErrorCategory = "AUTH"
	CategoryNotFound     ErrorCategory = "NOT_FOUND"
	CategoryConflict     ErrorCategory = "CONFLICT"
	CategoryUnauthorized ErrorCategory = "UNAUTHORIZED"
	CategoryInternal     ErrorCategory = "INTERNAL"
	CategoryExternal     ErrorCategory = "EXTERNAL"
)

type DomainError interface {
	error
	Code() string
	Category() ErrorCategory
	HTTPStatus() int
	Message() string
	Unwrap() error
	WithCause(cause error) DomainError
}

type domainError struct {
	code     string
	category ErrorCategory
	status   int
	message  string
	cause    error
}

func (e *domainError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

func (e *domainError) Code() string {
	return e.code
}

func (e *domainError) Category() ErrorCategory {
	return e.category
}

func (e *domainError) HTTPStatus() int {
	return e.status
}

func (e *domainError) Message() string {
	return e.message
}

func (e *domainError) Unwrap() error {
	return e.cause
}

func (e *domainError) WithCause(cause error) DomainError {
	return &domainError{
		code:     e.code,
		category: e.category,
		status:   e.status,
		message:  e.message,
		cause:    cause,
	}
}

func NewDomainError(code string, category ErrorCategory, status int, message string) DomainError {
	return &domainError{
		code:     code,
		category: category,
		status:   status,
		message:  message,
	}
}

func IsDomainError(err error) bool {
	var de DomainError
	return errors.As(err, &de)
}

func AsDomainError(err error) (DomainError, bool) {
	var de DomainError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}

var (
	ErrMissingRequiredEnv = NewDomainError(
		"MISSING_REQUIRED_ENV",
		CategoryValidation,
		http.StatusBadRequest,
		"missing required environment variable",
	)

	ErrInvalidJWTSecret = NewDomainError(
		"INVALID_JWT_SECRET",
		CategoryValidation,
		http.StatusInternalServerError,
		"JWT_SECRET must be at least 32 bytes",
	)

	ErrUserNotConnected = NewDomainError(
		"USER_NOT_CONNECTED",
		CategoryNotFound,
		http.StatusNotFound,
		"user not connected",
	)

	ErrEmptyQuery = NewDomainError(
		"EMPTY_QUERY",
		CategoryValidation,
		http.StatusBadRequest,
		"query is empty",
	)

	ErrIdentityKeyNotFound = NewDomainError(
		"IDENTITY_KEY_NOT_FOUND",
		CategoryNotFound,
		http.StatusNotFound,
		"identity key not found",
	)

	ErrInvalidPublicKey = NewDomainError(
		"INVALID_PUBLIC_KEY",
		CategoryValidation,
		http.StatusBadRequest,
		"invalid public key",
	)

	ErrTransferNotFound = NewDomainError(
		"TRANSFER_NOT_FOUND",
		CategoryNotFound,
		http.StatusNotFound,
		"transfer not found",
	)

	ErrTransferAlreadyExists = NewDomainError(
		"TRANSFER_ALREADY_EXISTS",
		CategoryConflict,
		http.StatusConflict,
		"transfer already exists",
	)

	ErrInvalidChunkIndex = NewDomainError(
		"INVALID_CHUNK_INDEX",
		CategoryValidation,
		http.StatusBadRequest,
		"invalid chunk index",
	)

	ErrUsernameAlreadyExists = NewDomainError(
		"USERNAME_ALREADY_EXISTS",
		CategoryConflict,
		http.StatusConflict,
		"username already exists",
	)

	ErrCircuitOpen = NewDomainError(
		"CIRCUIT_OPEN",
		CategoryExternal,
		http.StatusServiceUnavailable,
		"circuit breaker is open",
	)

	ErrInvalidToken = NewDomainError(
		"INVALID_TOKEN",
		CategoryUnauthorized,
		http.StatusUnauthorized,
		"token is not valid",
	)

	ErrEmptyUUID = NewDomainError(
		"EMPTY_UUID",
		CategoryValidation,
		http.StatusBadRequest,
		"uuid cannot be empty",
	)

	ErrUserNotFound = NewDomainError(
		"USER_NOT_FOUND",
		CategoryNotFound,
		http.StatusNotFound,
		"user not found",
	)

	ErrInternalError = NewDomainError(
		"INTERNAL_ERROR",
		CategoryInternal,
		http.StatusInternalServerError,
		"internal server error",
	)

	ErrMarshalError = NewDomainError(
		"MARSHAL_ERROR",
		CategoryInternal,
		http.StatusInternalServerError,
		"failed to marshal data",
	)

	ErrSendTimeout = NewDomainError(
		"SEND_TIMEOUT",
		CategoryExternal,
		http.StatusRequestTimeout,
		"send operation timed out",
	)

	ErrInvalidPayload = NewDomainError(
		"INVALID_PAYLOAD",
		CategoryValidation,
		http.StatusBadRequest,
		"invalid payload",
	)

	ErrFileSizeExceeded = NewDomainError(
		"FILE_SIZE_EXCEEDED",
		CategoryValidation,
		http.StatusBadRequest,
		"file size exceeds maximum",
	)

	ErrInvalidFileSize = NewDomainError(
		"INVALID_FILE_SIZE",
		CategoryValidation,
		http.StatusBadRequest,
		"invalid file size",
	)

	ErrInvalidTotalChunks = NewDomainError(
		"INVALID_TOTAL_CHUNKS",
		CategoryValidation,
		http.StatusBadRequest,
		"invalid total chunks",
	)

	ErrInvalidMimeType = NewDomainError(
		"INVALID_MIME_TYPE",
		CategoryValidation,
		http.StatusBadRequest,
		"invalid mime type",
	)

	ErrMimeTypeNotAllowed = NewDomainError(
		"MIME_TYPE_NOT_ALLOWED",
		CategoryValidation,
		http.StatusBadRequest,
		"mime type not allowed",
	)

	ErrUnknownMessageType = NewDomainError(
		"UNKNOWN_MESSAGE_TYPE",
		CategoryValidation,
		http.StatusBadRequest,
		"unknown message type",
	)

	ErrInvalidTokenSigningMethod = NewDomainError(
		"INVALID_TOKEN_SIGNING_METHOD",
		CategoryUnauthorized,
		http.StatusUnauthorized,
		"invalid token signing method",
	)

	ErrInvalidTokenClaims = NewDomainError(
		"INVALID_TOKEN_CLAIMS",
		CategoryUnauthorized,
		http.StatusUnauthorized,
		"invalid token claims",
	)

	ErrMissingTokenClaims = NewDomainError(
		"MISSING_TOKEN_CLAIMS",
		CategoryUnauthorized,
		http.StatusUnauthorized,
		"missing required token claims",
	)

	ErrDatabaseError = NewDomainError(
		"DATABASE_ERROR",
		CategoryInternal,
		http.StatusInternalServerError,
		"database operation failed",
	)

	ErrIdentityKeyCreateFailed = NewDomainError(
		"IDENTITY_KEY_CREATE_FAILED",
		CategoryInternal,
		http.StatusInternalServerError,
		"failed to create identity key",
	)

	ErrIdentityKeyGetFailed = NewDomainError(
		"IDENTITY_KEY_GET_FAILED",
		CategoryInternal,
		http.StatusInternalServerError,
		"failed to get identity key",
	)

	ErrFingerprintGetFailed = NewDomainError(
		"FINGERPRINT_GET_FAILED",
		CategoryInternal,
		http.StatusInternalServerError,
		"failed to get fingerprint",
	)

	ErrUserGetFailed = NewDomainError(
		"USER_GET_FAILED",
		CategoryInternal,
		http.StatusInternalServerError,
		"failed to get user",
	)

	ErrUserSearchFailed = NewDomainError(
		"USER_SEARCH_FAILED",
		CategoryInternal,
		http.StatusInternalServerError,
		"failed to search users",
	)
)
