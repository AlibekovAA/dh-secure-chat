package websocket

import (
	"fmt"
	"strings"
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
		return fmt.Errorf("file size %d exceeds maximum %d (audio=%v)", p.TotalSize, maxSize, isAudio)
	}

	if p.TotalSize <= 0 {
		return fmt.Errorf("invalid file size %d", p.TotalSize)
	}

	if p.TotalChunks <= 0 || p.TotalChunks > 1000 {
		return fmt.Errorf("invalid total_chunks %d", p.TotalChunks)
	}

	if isAudio {
		if !isValidAudioMimeType(p.MimeType) {
			return fmt.Errorf("invalid audio mime type %s", p.MimeType)
		}
	} else {
		if !v.allowedMimeTypes[p.MimeType] {
			return fmt.Errorf("mime type not allowed: %s", p.MimeType)
		}
	}

	return nil
}
