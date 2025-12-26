package transfer

import (
	"sync"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

type TrackRequest struct {
	FileID      string
	From        string
	To          string
	TotalChunks int
}

type Transfer struct {
	FileID         string
	From           string
	To             string
	StartedAt      time.Time
	LastChunkAt    time.Time
	ReceivedChunks int
	TotalChunks    int
}

type Tracker interface {
	Track(req TrackRequest) error
	UpdateProgress(fileID string, chunkIndex int) error
	Complete(fileID string) error
	GetTransfersForUser(userID string) []*Transfer
	CleanupStale() int
}

type InMemoryTracker struct {
	transfers sync.Map
	timeout   time.Duration
	clock     clock.Clock
}

func NewTracker(timeout time.Duration, clock clock.Clock) Tracker {
	return &InMemoryTracker{
		timeout: timeout,
		clock:   clock,
	}
}

func (t *InMemoryTracker) Track(req TrackRequest) error {
	now := t.clock.Now()

	transfer := &Transfer{
		FileID:         req.FileID,
		From:           req.From,
		To:             req.To,
		StartedAt:      now,
		LastChunkAt:    now,
		TotalChunks:    req.TotalChunks,
		ReceivedChunks: 0,
	}

	if _, loaded := t.transfers.LoadOrStore(req.FileID, transfer); loaded {
		return commonerrors.ErrTransferAlreadyExists
	}

	return nil
}

func (t *InMemoryTracker) UpdateProgress(fileID string, chunkIndex int) error {
	value, ok := t.transfers.Load(fileID)
	if !ok {
		return commonerrors.ErrTransferNotFound
	}

	if chunkIndex < 0 {
		return commonerrors.ErrInvalidChunkIndex
	}

	transfer := value.(*Transfer)
	transfer.LastChunkAt = t.clock.Now()
	if chunkIndex >= transfer.TotalChunks {
		return commonerrors.ErrInvalidChunkIndex
	}
	if chunkIndex >= transfer.ReceivedChunks {
		transfer.ReceivedChunks = chunkIndex + 1
	}

	return nil
}

func (t *InMemoryTracker) Complete(fileID string) error {
	if _, ok := t.transfers.Load(fileID); !ok {
		return commonerrors.ErrTransferNotFound
	}

	t.transfers.Delete(fileID)
	return nil
}

func (t *InMemoryTracker) GetTransfersForUser(userID string) []*Transfer {
	var result []*Transfer

	t.transfers.Range(func(key, value interface{}) bool {
		transfer := value.(*Transfer)
		if transfer.From == userID || transfer.To == userID {
			copy := *transfer
			result = append(result, &copy)
		}
		return true
	})

	return result
}

func (t *InMemoryTracker) CleanupStale() int {
	now := t.clock.Now()
	removed := 0

	t.transfers.Range(func(key, value interface{}) bool {
		transfer := value.(*Transfer)
		if now.Sub(transfer.LastChunkAt) > t.timeout {
			t.transfers.Delete(key)
			removed++
		}
		return true
	})

	return removed
}
