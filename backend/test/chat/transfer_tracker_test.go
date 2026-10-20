package chat

import (
	"errors"
	"testing"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/transfer"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

func TestInMemoryTrackerValidate(t *testing.T) {
	tracker := transfer.NewTracker(time.Minute, clock.NewMockClock(time.Now()))
	req := transfer.TrackRequest{
		FileID:      "file-1",
		From:        "sender-1",
		To:          "recipient-1",
		TotalSize:   2,
		TotalChunks: 1,
		ChunkSize:   2,
	}
	if err := tracker.Track(req); err != nil {
		t.Fatalf("track transfer: %v", err)
	}

	tests := []struct {
		name        string
		fileID      string
		senderID    string
		recipientID string
		wantErr     bool
	}{
		{
			name:        "matching participants",
			fileID:      req.FileID,
			senderID:    req.From,
			recipientID: req.To,
		},
		{
			name:        "missing transfer",
			fileID:      "missing-file",
			senderID:    req.From,
			recipientID: req.To,
			wantErr:     true,
		},
		{
			name:        "different sender",
			fileID:      req.FileID,
			senderID:    "sender-2",
			recipientID: req.To,
			wantErr:     true,
		},
		{
			name:        "different recipient",
			fileID:      req.FileID,
			senderID:    req.From,
			recipientID: "recipient-2",
			wantErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := tracker.Validate(test.fileID, test.senderID, test.recipientID)
			if !test.wantErr && err != nil {
				t.Fatalf("validate transfer: %v", err)
			}
			if test.wantErr && !errors.Is(err, commonerrors.ErrTransferNotFound) {
				t.Fatalf("expected ErrTransferNotFound, got %v", err)
			}
		})
	}
}

func TestInMemoryTrackerRejectsIncompleteBinding(t *testing.T) {
	tracker := transfer.NewTracker(time.Minute, clock.NewMockClock(time.Now()))
	tests := []transfer.TrackRequest{
		{From: "sender-1", To: "recipient-1", TotalSize: 1, TotalChunks: 1, ChunkSize: 1},
		{FileID: "file-1", To: "recipient-1", TotalSize: 1, TotalChunks: 1, ChunkSize: 1},
		{FileID: "file-1", From: "sender-1", TotalSize: 1, TotalChunks: 1, ChunkSize: 1},
	}

	for _, req := range tests {
		if err := tracker.Track(req); !errors.Is(err, commonerrors.ErrInvalidPayload) {
			t.Fatalf("expected ErrInvalidPayload, got %v", err)
		}
	}
}

func TestInMemoryTrackerAcceptsValidChunksAndFinalizes(t *testing.T) {
	tracker := transfer.NewTracker(time.Minute, clock.NewMockClock(time.Now()))
	req := transfer.TrackRequest{
		FileID:      "file-1",
		From:        "sender-1",
		To:          "recipient-1",
		TotalSize:   3,
		TotalChunks: 2,
		ChunkSize:   2,
	}
	if err := tracker.Track(req); err != nil {
		t.Fatalf("track transfer: %v", err)
	}

	chunks := []transfer.ChunkRequest{
		{FileID: req.FileID, SenderID: req.From, RecipientID: req.To, ChunkIndex: 1, TotalChunks: 2, Size: 1},
		{FileID: req.FileID, SenderID: req.From, RecipientID: req.To, ChunkIndex: 0, TotalChunks: 2, Size: 2},
	}
	for _, chunk := range chunks {
		if err := tracker.AcceptChunk(chunk); err != nil {
			t.Fatalf("accept chunk %d: %v", chunk.ChunkIndex, err)
		}
	}

	tracked, ok := tracker.GetTransferByID(req.FileID)
	if !ok {
		t.Fatal("expected tracked transfer")
	}
	if tracked.ReceivedChunks != 2 || tracked.ReceivedSize != 3 {
		t.Fatalf("unexpected progress: chunks=%d size=%d", tracked.ReceivedChunks, tracked.ReceivedSize)
	}

	if err := tracker.Finalize(req.FileID, req.From, req.To); err != nil {
		t.Fatalf("finalize transfer: %v", err)
	}
	if _, ok := tracker.GetTransferByID(req.FileID); ok {
		t.Fatal("expected finalized transfer to be removed")
	}
}

func TestInMemoryTrackerRejectsInvalidChunkAndAborts(t *testing.T) {
	tests := []struct {
		name     string
		first    *transfer.ChunkRequest
		chunk    transfer.ChunkRequest
		expected error
	}{
		{
			name:     "total chunks mismatch",
			chunk:    transfer.ChunkRequest{ChunkIndex: 0, TotalChunks: 3, Size: 2},
			expected: commonerrors.ErrInvalidTotalChunks,
		},
		{
			name:     "index out of range",
			chunk:    transfer.ChunkRequest{ChunkIndex: 2, TotalChunks: 2, Size: 1},
			expected: commonerrors.ErrInvalidChunkIndex,
		},
		{
			name:     "wrong size",
			chunk:    transfer.ChunkRequest{ChunkIndex: 0, TotalChunks: 2, Size: 1},
			expected: commonerrors.ErrInvalidChunkSize,
		},
		{
			name:     "duplicate index",
			first:    &transfer.ChunkRequest{ChunkIndex: 0, TotalChunks: 2, Size: 2},
			chunk:    transfer.ChunkRequest{ChunkIndex: 0, TotalChunks: 2, Size: 2},
			expected: commonerrors.ErrInvalidChunkIndex,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tracker := transfer.NewTracker(time.Minute, clock.NewMockClock(time.Now()))
			req := transfer.TrackRequest{
				FileID:      "file-1",
				From:        "sender-1",
				To:          "recipient-1",
				TotalSize:   3,
				TotalChunks: 2,
				ChunkSize:   2,
			}
			if err := tracker.Track(req); err != nil {
				t.Fatalf("track transfer: %v", err)
			}

			prepare := func(chunk *transfer.ChunkRequest) {
				chunk.FileID = req.FileID
				chunk.SenderID = req.From
				chunk.RecipientID = req.To
			}
			if test.first != nil {
				prepare(test.first)
				if err := tracker.AcceptChunk(*test.first); err != nil {
					t.Fatalf("accept initial chunk: %v", err)
				}
			}
			prepare(&test.chunk)
			if err := tracker.AcceptChunk(test.chunk); !errors.Is(err, test.expected) {
				t.Fatalf("expected %v, got %v", test.expected, err)
			}
			if _, ok := tracker.GetTransferByID(req.FileID); ok {
				t.Fatal("expected invalid transfer to be removed")
			}
		})
	}
}

func TestInMemoryTrackerLimitsConcurrentTransfersPerUser(t *testing.T) {
	tracker := transfer.NewTracker(time.Minute, clock.NewMockClock(time.Now()))
	for i := 0; i < 5; i++ {
		req := transfer.TrackRequest{
			FileID:      string(rune('a' + i)),
			From:        "sender-1",
			To:          string(rune('k' + i)),
			TotalSize:   1,
			TotalChunks: 1,
			ChunkSize:   1,
		}
		if err := tracker.Track(req); err != nil {
			t.Fatalf("track transfer %d: %v", i, err)
		}
	}

	err := tracker.Track(transfer.TrackRequest{
		FileID:      "sixth",
		From:        "sender-1",
		To:          "recipient-6",
		TotalSize:   1,
		TotalChunks: 1,
		ChunkSize:   1,
	})
	if !errors.Is(err, commonerrors.ErrTransferLimitExceeded) {
		t.Fatalf("expected ErrTransferLimitExceeded, got %v", err)
	}
}

func TestInMemoryTrackerReleasesUserSlotOnComplete(t *testing.T) {
	tracker := transfer.NewTracker(time.Minute, clock.NewMockClock(time.Now()))
	for i := 0; i < 5; i++ {
		req := transfer.TrackRequest{
			FileID:      string(rune('a' + i)),
			From:        "sender-1",
			To:          string(rune('k' + i)),
			TotalSize:   1,
			TotalChunks: 1,
			ChunkSize:   1,
		}
		if err := tracker.Track(req); err != nil {
			t.Fatalf("track transfer %d: %v", i, err)
		}
	}

	if err := tracker.Complete("a"); err != nil {
		t.Fatalf("complete transfer: %v", err)
	}
	if transfers := tracker.GetTransfersForUser("sender-1"); len(transfers) != 4 {
		t.Fatalf("expected 4 active transfers, got %d", len(transfers))
	}
	if err := tracker.Track(transfer.TrackRequest{
		FileID:      "replacement",
		From:        "sender-1",
		To:          "recipient-6",
		TotalSize:   1,
		TotalChunks: 1,
		ChunkSize:   1,
	}); err != nil {
		t.Fatalf("track replacement transfer: %v", err)
	}
}

func TestInMemoryTrackerOwnershipMismatchDoesNotAbortTransfer(t *testing.T) {
	tracker := transfer.NewTracker(time.Minute, clock.NewMockClock(time.Now()))
	req := transfer.TrackRequest{
		FileID:      "file-1",
		From:        "sender-1",
		To:          "recipient-1",
		TotalSize:   1,
		TotalChunks: 1,
		ChunkSize:   1,
	}
	if err := tracker.Track(req); err != nil {
		t.Fatalf("track transfer: %v", err)
	}

	err := tracker.AcceptChunk(transfer.ChunkRequest{
		FileID:      req.FileID,
		SenderID:    "sender-2",
		RecipientID: req.To,
		ChunkIndex:  0,
		TotalChunks: 1,
		Size:        1,
	})
	if !errors.Is(err, commonerrors.ErrTransferNotFound) {
		t.Fatalf("expected ErrTransferNotFound, got %v", err)
	}
	if err := tracker.Validate(req.FileID, req.From, req.To); err != nil {
		t.Fatalf("expected original transfer to remain valid: %v", err)
	}
}

func TestInMemoryTrackerIncompleteFinalizeAbortsTransfer(t *testing.T) {
	tracker := transfer.NewTracker(time.Minute, clock.NewMockClock(time.Now()))
	req := transfer.TrackRequest{
		FileID:      "file-1",
		From:        "sender-1",
		To:          "recipient-1",
		TotalSize:   2,
		TotalChunks: 2,
		ChunkSize:   1,
	}
	if err := tracker.Track(req); err != nil {
		t.Fatalf("track transfer: %v", err)
	}

	if err := tracker.Finalize(req.FileID, req.From, req.To); !errors.Is(err, commonerrors.ErrInvalidTotalChunks) {
		t.Fatalf("expected ErrInvalidTotalChunks, got %v", err)
	}
	if _, ok := tracker.GetTransferByID(req.FileID); ok {
		t.Fatal("expected incomplete transfer to be removed")
	}
}
