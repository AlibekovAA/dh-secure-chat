package http

const (
	CodeUnknown                  = "UNKNOWN"
	CodeMethodNotAllowed         = "METHOD_NOT_ALLOWED"
	CodeInvalidJSON              = "INVALID_JSON"
	CodeBadRequest               = "BAD_REQUEST"
	CodeInvalidPath              = "INVALID_PATH"
	CodeUserIDRequired           = "USER_ID_REQUIRED"
	CodeInvalidUserIDFormat      = "INVALID_USER_ID_FORMAT"
	CodeInvalidIdentityPubKeyEnc = "INVALID_IDENTITY_PUB_KEY_ENCODING"
	CodeMissingRefreshToken      = "MISSING_REFRESH_TOKEN"
	CodeMissingAuthorization     = "MISSING_AUTHORIZATION"
	CodeInvalidToken             = "INVALID_TOKEN"
	CodeTokenMissingJTI          = "TOKEN_MISSING_JTI"
)
