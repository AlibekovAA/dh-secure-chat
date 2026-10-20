package transfer

import (
	"sync"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

type TrackRequest struct {
	FileID      string
	From        string
	To          string
	TotalSize   int64
	TotalChunks int
	ChunkSize   int
}

type ChunkRequest struct {
	FileID      string
	SenderID    string
	RecipientID string
	ChunkIndex  int
	TotalChunks int
	Size        int64
}

type Transfer struct {
	FileID         string
	From           string
	To             string
	StartedAt      time.Time
	LastChunkAt    time.Time
	ReceivedChunks int
	ReceivedSize   int64
	TotalSize      int64
	TotalChunks    int
	ChunkSize      int
	received       map[int]struct{}
}

type Tracker interface {
	Track(req TrackRequest) error
	Validate(fileID string, senderID string, recipientID string) error
	AcceptChunk(req ChunkRequest) error
	Finalize(fileID string, senderID string, recipientID string) error
	Abort(fileID string, senderID string, recipientID string) error
	Complete(fileID string) error
	GetTransfersForUser(userID string) []*Transfer
	GetTransferByID(fileID string) (*Transfer, bool)
	CleanupStale() int
}

type InMemoryTracker struct {
	mu              sync.RWMutex
	transfers       map[string]*Transfer
	transfersByUser map[string]map[string]struct{}
	timeout         time.Duration
	clock           clock.Clock
}

func (t *InMemoryTracker) GetTransferByID(fileID string) (*Transfer, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	transfer, ok := t.transfers[fileID]
	if !ok {
		return nil, false
	}
	copy := *transfer
	copy.received = nil
	return &copy, true
}

func NewTracker(timeout time.Duration, clock clock.Clock) Tracker {
	return &InMemoryTracker{
		transfers:       make(map[string]*Transfer),
		transfersByUser: make(map[string]map[string]struct{}),
		timeout:         timeout,
		clock:           clock,
	}
}

func (t *InMemoryTracker) Track(req TrackRequest) error {
	if req.FileID == "" || req.From == "" || req.To == "" || req.From == req.To {
		return commonerrors.ErrInvalidPayload
	}
	if req.TotalSize <= 0 {
		return commonerrors.ErrInvalidFileSize
	}
	if req.ChunkSize <= 0 || req.ChunkSize > constants.MaxFileChunkSizeBytes {
		return commonerrors.ErrInvalidChunkSize
	}
	expectedChunks := (req.TotalSize + int64(req.ChunkSize) - 1) / int64(req.ChunkSize)
	if req.TotalChunks <= 0 || req.TotalChunks > constants.MaxFileChunks || int64(req.TotalChunks) != expectedChunks {
		return commonerrors.ErrInvalidTotalChunks
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.transfers[req.FileID]; exists {
		return commonerrors.ErrTransferAlreadyExists
	}
	if len(t.transfersByUser[req.From]) >= constants.MaxConcurrentFileTransfersPerUser ||
		len(t.transfersByUser[req.To]) >= constants.MaxConcurrentFileTransfersPerUser {
		return commonerrors.ErrTransferLimitExceeded
	}

	now := t.clock.Now()

	transfer := &Transfer{
		FileID:         req.FileID,
		From:           req.From,
		To:             req.To,
		StartedAt:      now,
		LastChunkAt:    now,
		TotalSize:      req.TotalSize,
		TotalChunks:    req.TotalChunks,
		ChunkSize:      req.ChunkSize,
		ReceivedChunks: 0,
		ReceivedSize:   0,
		received:       make(map[int]struct{}, req.TotalChunks),
	}
	t.transfers[req.FileID] = transfer
	t.addUserTransfer(req.From, req.FileID)
	t.addUserTransfer(req.To, req.FileID)

	return nil
}

func (t *InMemoryTracker) Validate(fileID string, senderID string, recipientID string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	transfer, ok := t.transfers[fileID]
	if !ok {
		return commonerrors.ErrTransferNotFound
	}
	if transfer.From != senderID || transfer.To != recipientID {
		return commonerrors.ErrTransferNotFound
	}

	return nil
}

func (t *InMemoryTracker) AcceptChunk(req ChunkRequest) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	transfer, ok := t.transfers[req.FileID]
	if !ok {
		return commonerrors.ErrTransferNotFound
	}
	if transfer.From != req.SenderID || transfer.To != req.RecipientID {
		return commonerrors.ErrTransferNotFound
	}
	if req.TotalChunks != transfer.TotalChunks {
		return t.abortInvalidChunk(req.FileID, commonerrors.ErrInvalidTotalChunks)
	}
	if req.ChunkIndex < 0 || req.ChunkIndex >= transfer.TotalChunks {
		return t.abortInvalidChunk(req.FileID, commonerrors.ErrInvalidChunkIndex)
	}
	if _, exists := transfer.received[req.ChunkIndex]; exists {
		return t.abortInvalidChunk(req.FileID, commonerrors.ErrInvalidChunkIndex)
	}

	expectedSize := int64(transfer.ChunkSize)
	if req.ChunkIndex == transfer.TotalChunks-1 {
		expectedSize = transfer.TotalSize - int64(transfer.ChunkSize)*int64(transfer.TotalChunks-1)
	}
	if req.Size != expectedSize || transfer.ReceivedSize+req.Size > transfer.TotalSize {
		return t.abortInvalidChunk(req.FileID, commonerrors.ErrInvalidChunkSize)
	}

	transfer.received[req.ChunkIndex] = struct{}{}
	transfer.ReceivedChunks++
	transfer.ReceivedSize += req.Size
	transfer.LastChunkAt = t.clock.Now()
	return nil
}

func (t *InMemoryTracker) Finalize(fileID string, senderID string, recipientID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	transfer, ok := t.transfers[fileID]
	if !ok || transfer.From != senderID || transfer.To != recipientID {
		return commonerrors.ErrTransferNotFound
	}
	if transfer.ReceivedChunks != transfer.TotalChunks {
		t.deleteTransfer(fileID)
		return commonerrors.ErrInvalidTotalChunks
	}
	if transfer.ReceivedSize != transfer.TotalSize {
		t.deleteTransfer(fileID)
		return commonerrors.ErrInvalidFileSize
	}

	t.deleteTransfer(fileID)
	return nil
}

func (t *InMemoryTracker) Abort(fileID string, senderID string, recipientID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	transfer, ok := t.transfers[fileID]
	if !ok || transfer.From != senderID || transfer.To != recipientID {
		return commonerrors.ErrTransferNotFound
	}
	t.deleteTransfer(fileID)
	return nil
}

func (t *InMemoryTracker) Complete(fileID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.transfers[fileID]; !ok {
		return commonerrors.ErrTransferNotFound
	}

	t.deleteTransfer(fileID)
	return nil
}

func (t *InMemoryTracker) GetTransfersForUser(userID string) []*Transfer {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*Transfer, 0, len(t.transfersByUser[userID]))
	for fileID := range t.transfersByUser[userID] {
		transfer := t.transfers[fileID]
		copy := *transfer
		copy.received = nil
		result = append(result, &copy)
	}

	return result
}

func (t *InMemoryTracker) CleanupStale() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.clock.Now()
	removed := 0
	for fileID, transfer := range t.transfers {
		if now.Sub(transfer.LastChunkAt) > t.timeout {
			t.deleteTransfer(fileID)
			removed++
		}
	}

	return removed
}

func (t *InMemoryTracker) addUserTransfer(userID string, fileID string) {
	if t.transfersByUser[userID] == nil {
		t.transfersByUser[userID] = make(map[string]struct{})
	}
	t.transfersByUser[userID][fileID] = struct{}{}
}

func (t *InMemoryTracker) deleteTransfer(fileID string) {
	transfer, ok := t.transfers[fileID]
	if !ok {
		return
	}
	delete(t.transfers, fileID)
	for _, userID := range []string{transfer.From, transfer.To} {
		delete(t.transfersByUser[userID], fileID)
		if len(t.transfersByUser[userID]) == 0 {
			delete(t.transfersByUser, userID)
		}
	}
}

func (t *InMemoryTracker) abortInvalidChunk(fileID string, err error) error {
	t.deleteTransfer(fileID)
	return err
}
