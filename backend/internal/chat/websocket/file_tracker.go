package websocket

import (
	"sync"
	"time"
)

type FileTransfer struct {
	FileID         string
	From           string
	To             string
	StartedAt      time.Time
	LastChunkAt    time.Time
	ReceivedChunks int
	TotalChunks    int
}

type FileTransferTracker struct {
	transfers sync.Map
	timeout   time.Duration
}

func NewFileTransferTracker(timeout time.Duration) *FileTransferTracker {
	return &FileTransferTracker{
		timeout: timeout,
	}
}

func (t *FileTransferTracker) Track(payload FileStartPayload) {
	transfer := &FileTransfer{
		FileID:         payload.FileID,
		From:           payload.From,
		To:             payload.To,
		StartedAt:      time.Now(),
		LastChunkAt:    time.Now(),
		TotalChunks:    payload.TotalChunks,
		ReceivedChunks: 0,
	}
	t.transfers.Store(payload.FileID, transfer)
}

func (t *FileTransferTracker) UpdateProgress(fileID string, chunkIndex int) {
	value, ok := t.transfers.Load(fileID)
	if !ok {
		return
	}

	transfer := value.(*FileTransfer)
	transfer.LastChunkAt = time.Now()
	if chunkIndex > transfer.ReceivedChunks {
		transfer.ReceivedChunks = chunkIndex + 1
	}
}

func (t *FileTransferTracker) Complete(fileID string) {
	t.transfers.Delete(fileID)
}

func (t *FileTransferTracker) GetTransfersForUser(userID string) []*FileTransfer {
	var result []*FileTransfer
	t.transfers.Range(func(key, value interface{}) bool {
		transfer := value.(*FileTransfer)
		if transfer.From == userID || transfer.To == userID {
			result = append(result, transfer)
		}
		return true
	})
	return result
}

func (t *FileTransferTracker) CleanupStale() {
	now := time.Now()
	t.transfers.Range(func(key, value interface{}) bool {
		transfer := value.(*FileTransfer)
		if now.Sub(transfer.LastChunkAt) > t.timeout {
			t.transfers.Delete(key)
		}
		return true
	})
}
