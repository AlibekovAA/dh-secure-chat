package websocket

import (
	"strings"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

type MessageValidator interface {
	ValidateFileStart(p FileStartPayload) error
}

type DefaultValidator struct {
	maxFileSize      int64
	maxVoiceSize     int64
	allowedMimeTypes map[string]bool
}

func NewDefaultValidator(maxFileSize, maxVoiceSize int64) *DefaultValidator {
	allowedMimeTypes := map[string]bool{
		"image/jpeg":         true,
		"image/jpg":          true,
		"image/png":          true,
		"image/gif":          true,
		"image/webp":         true,
		"image/svg+xml":      true,
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
		"application/vnd.ms-powerpoint":                                             true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		"text/plain":                   true,
		"text/csv":                     true,
		"text/markdown":                true,
		"application/zip":              true,
		"application/x-rar-compressed": true,
		"application/x-tar":            true,
		"application/gzip":             true,
		"application/octet-stream":     true,
		"video/mp4":                    true,
		"video/webm":                   true,
		"video/ogg":                    true,
		"video/quicktime":              true,
		"video/x-msvideo":              true,
		"video/x-matroska":             true,
	}

	return &DefaultValidator{
		maxFileSize:      maxFileSize,
		maxVoiceSize:     maxVoiceSize,
		allowedMimeTypes: allowedMimeTypes,
	}
}

func (v *DefaultValidator) ValidateFileStart(p FileStartPayload) error {
	isAudio := strings.HasPrefix(p.MimeType, "audio/")
	maxSize := v.maxFileSize
	if isAudio {
		maxSize = v.maxVoiceSize
	}

	if p.TotalSize > maxSize {
		return commonerrors.ErrFileSizeExceeded
	}

	if p.TotalSize <= 0 {
		return commonerrors.ErrInvalidFileSize
	}

	if p.TotalChunks <= 0 || p.TotalChunks > 1000 {
		return commonerrors.ErrInvalidTotalChunks
	}

	if isAudio {
		if !isValidAudioMimeType(p.MimeType) {
			return commonerrors.ErrInvalidMimeType
		}
	} else {
		if !v.allowedMimeTypes[p.MimeType] {
			return commonerrors.ErrMimeTypeNotAllowed
		}
	}

	return nil
}
