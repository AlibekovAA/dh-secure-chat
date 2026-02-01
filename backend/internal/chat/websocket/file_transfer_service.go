package websocket

import (
	"context"
	"errors"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/chat/transfer"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	observabilitymetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

type FileTransferService struct {
	tracker transfer.Tracker
	sender  MessageSender
	log     *logger.Logger
	clock   clock.Clock
	ctx     context.Context
}

func NewFileTransferService(sender MessageSender, timeout time.Duration, clk clock.Clock, log *logger.Logger, ctx context.Context) *FileTransferService {
	return &FileTransferService{
		tracker: transfer.NewTracker(timeout, clk),
		sender:  sender,
		log:     log,
		clock:   clk,
		ctx:     ctx,
	}
}

func (s *FileTransferService) Track(payload FileStartPayload) {
	req := transfer.TrackRequest{
		FileID:      payload.FileID,
		From:        payload.From,
		To:          payload.To,
		TotalChunks: payload.TotalChunks,
	}

	if err := s.tracker.Track(req); err != nil {
		s.log.WithFields(s.ctx, logger.Fields{
			"file_id": payload.FileID,
			"from":    payload.From,
			"to":      payload.To,
			"action":  "ws_file_track",
		}).Warnf("websocket failed to track file transfer: %v", err)
		observabilitymetrics.ChatWebSocketFileTransferFailures.WithLabelValues("track_failed").Inc()
	}
}

func (s *FileTransferService) UpdateProgress(fileID string, chunkIndex int) {
	if err := s.tracker.UpdateProgress(fileID, chunkIndex); err != nil {
		if errors.Is(err, commonerrors.ErrTransferNotFound) {
			if s.log.ShouldLog(logger.DEBUG) {
				s.log.WithFields(s.ctx, logger.Fields{
					"file_id":     fileID,
					"chunk_index": chunkIndex,
					"action":      "ws_file_progress_skipped",
				}).Debug("websocket file transfer progress skipped (no tracking)")
			}
			return
		}
		s.log.WithFields(s.ctx, logger.Fields{
			"file_id":     fileID,
			"chunk_index": chunkIndex,
			"action":      "ws_file_progress_failed",
		}).Warnf("websocket file transfer progress failed: %v", err)
	}
}

func (s *FileTransferService) Complete(fileID string) {
	if err := s.tracker.Complete(fileID); err != nil {
		if errors.Is(err, commonerrors.ErrTransferNotFound) {
			if s.log.ShouldLog(logger.DEBUG) {
				s.log.WithFields(s.ctx, logger.Fields{
					"file_id": fileID,
					"action":  "ws_file_complete_skipped",
				}).Debug("websocket file transfer complete skipped (no tracking)")
			}
			return
		}
		s.log.WithFields(s.ctx, logger.Fields{
			"file_id": fileID,
			"action":  "ws_file_complete_failed",
		}).Warnf("websocket failed to complete file transfer: %v", err)
		observabilitymetrics.ChatWebSocketFileTransferFailures.WithLabelValues("complete_failed").Inc()
	}
}

func (s *FileTransferService) NotifyFailed(tr *transfer.Transfer) {
	if tr.To == "" {
		return
	}

	observabilitymetrics.ChatWebSocketFileTransferFailures.WithLabelValues("timeout_or_disconnect").Inc()

	msg, err := marshalMessage(TypeFileComplete, FileCompletePayload{
		To:     tr.To,
		From:   tr.From,
		FileID: tr.FileID,
	})
	if err != nil {
		s.log.WithFields(s.ctx, logger.Fields{
			"file_id": tr.FileID,
			"action":  "ws_file_failed_marshal",
		}).Errorf("websocket failed to marshal file_failed: %v", err)
		return
	}
	if err := s.sender.SendToUserWithContext(s.ctx, tr.To, msg); err != nil {
		s.log.WithFields(s.ctx, logger.Fields{
			"to":      tr.To,
			"file_id": tr.FileID,
			"action":  "ws_file_failed_notify",
		}).Warnf("websocket failed to notify file transfer failure: %v", err)
	}
}

func (s *FileTransferService) OnUserDisconnected(userID string) {
	transfers := s.tracker.GetTransfersForUser(userID)
	for _, tr := range transfers {
		s.NotifyFailed(tr)
		if err := s.tracker.Complete(tr.FileID); err != nil {
			s.log.WithFields(s.ctx, logger.Fields{
				"user_id": userID,
				"file_id": tr.FileID,
				"action":  "ws_file_complete_on_unregister",
			}).Warnf("websocket failed to complete file transfer on unregister: %v", err)
		}
	}
}

func (s *FileTransferService) StartCleanup() {
	ticker := time.NewTicker(constants.WebSocketFileTrackerCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if removed := s.tracker.CleanupStale(); removed > 0 {
				s.log.Debugf("websocket cleaned up stale file transfers count=%d", removed)
			}
		}
	}
}
