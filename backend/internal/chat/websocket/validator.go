package websocket

import (
	"fmt"
	"strings"
)

type MessageValidator interface {
	ValidateFileStart(p FileStartPayload) error
	ValidateAudio(mimeType string, size int64) error
}

type DefaultValidator struct {
	maxFileSize  int64
	maxVoiceSize int64
}

func NewDefaultValidator(maxFileSize, maxVoiceSize int64) *DefaultValidator {
	return &DefaultValidator{
		maxFileSize:  maxFileSize,
		maxVoiceSize: maxVoiceSize,
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
	}

	return nil
}

func (v *DefaultValidator) ValidateAudio(mimeType string, size int64) error {
	if !isValidAudioMimeType(mimeType) {
		return fmt.Errorf("invalid audio mime type %s", mimeType)
	}

	if size > v.maxVoiceSize {
		return fmt.Errorf("audio size %d exceeds maximum %d", size, v.maxVoiceSize)
	}

	return nil
}
