package websocket

import (
	"context"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	prommetrics "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/prometheus"
)

type messageTask struct {
	client *Client
	msg    *WSMessage
	ctx    context.Context
}

type MessageProcessor struct {
	workers   int
	queue     chan messageTask
	router    MessageRouter
	log       *logger.Logger
	queueSize int
}

func NewMessageProcessor(workers int, router MessageRouter, log *logger.Logger, queueSize int) *MessageProcessor {
	if queueSize <= 0 {
		queueSize = 1000
	}

	p := &MessageProcessor{
		workers:   workers,
		queue:     make(chan messageTask, queueSize),
		router:    router,
		log:       log,
		queueSize: queueSize,
	}

	for i := 0; i < workers; i++ {
		go p.worker()
	}

	return p
}

func (p *MessageProcessor) worker() {
	for task := range p.queue {
		p.process(task.ctx, task.client, task.msg)
	}
}

func (p *MessageProcessor) process(ctx context.Context, client *Client, msg *WSMessage) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := p.router.Route(ctx, client, msg); err != nil {
		p.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"type":    string(msg.Type),
			"action":  "ws_message_processing_failed",
		}).Warnf("websocket message processing failed: %v", err)
	}

	duration := time.Since(start).Seconds()
	prommetrics.ChatWebSocketMessageProcessingDurationSeconds.WithLabelValues(string(msg.Type)).Observe(duration)
}

func (p *MessageProcessor) Submit(ctx context.Context, client *Client, msg *WSMessage) {
	task := messageTask{
		client: client,
		msg:    msg,
		ctx:    ctx,
	}

	select {
	case p.queue <- task:
		prommetrics.ChatWebSocketMessageProcessorQueueSize.Set(float64(len(p.queue)))
	default:
		p.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"type":    string(msg.Type),
			"action":  "ws_queue_full",
		}).Warn("websocket message queue full")
		prommetrics.ChatWebSocketMessageProcessorQueueSize.Set(float64(len(p.queue)))
	}
}

func (p *MessageProcessor) Shutdown() {
	close(p.queue)
}
