package websocket

import (
	"context"
	"errors"
	"hash/fnv"
	"sync"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/constants"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/observability/metrics"
)

type messageTask struct {
	client *Client
	msg    *WSMessage
	ctx    context.Context
}

type MessageProcessor struct {
	queues []chan messageTask
	router MessageRouter
	log    *logger.Logger
	mu     sync.RWMutex
	closed bool
}

var (
	ErrProcessorQueueFull = errors.New("websocket message processor queue full")
	ErrProcessorClosed    = errors.New("websocket message processor is closed")
)

func NewMessageProcessor(workers int, router MessageRouter, log *logger.Logger, queueSize int) *MessageProcessor {
	if queueSize <= 0 {
		queueSize = constants.WebSocketProcessorDefaultQueueSize
	}

	if workers <= 0 {
		workers = 1
	}
	perQueueSize := (queueSize + workers - 1) / workers
	p := &MessageProcessor{
		queues: make([]chan messageTask, workers),
		router: router,
		log:    log,
	}

	for i := 0; i < workers; i++ {
		p.queues[i] = make(chan messageTask, perQueueSize)
		go p.worker(p.queues[i])
	}

	return p
}

func (p *MessageProcessor) worker(queue <-chan messageTask) {
	for task := range queue {
		metrics.ChatWebSocketMessageProcessorQueueSize.Set(float64(len(queue)))
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					p.log.Errorf("websocket message processor task panic: %v", recovered)
				}
			}()
			p.process(task.ctx, task.client, task.msg)
		}()
	}
}

func (p *MessageProcessor) process(ctx context.Context, client *Client, msg *WSMessage) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, constants.WebSocketProcessorTimeout)
	defer cancel()

	if err := p.router.Route(ctx, client, msg); err != nil {
		p.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"type":    string(msg.Type),
			"action":  "ws_message_processing_failed",
		}).Warnf("websocket message processing failed: %v", err)
	}

	duration := time.Since(start).Seconds()
	metrics.ChatWebSocketMessageProcessingDurationSeconds.WithLabelValues(string(msg.Type)).Observe(duration)
}

func (p *MessageProcessor) Submit(ctx context.Context, client *Client, msg *WSMessage) error {
	task := messageTask{
		client: client,
		msg:    msg,
		ctx:    ctx,
	}

	queue := p.queueFor(client.userID)
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return ErrProcessorClosed
	}
	select {
	case queue <- task:
		metrics.ChatWebSocketMessageProcessorQueueSize.Set(float64(len(queue)))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		p.log.WithFields(ctx, logger.Fields{
			"user_id": client.userID,
			"type":    string(msg.Type),
			"action":  "ws_queue_full",
		}).Warn("websocket message queue full")
		metrics.ChatWebSocketMessageProcessorQueueSize.Set(float64(len(queue)))
		return ErrProcessorQueueFull
	}
}

func (p *MessageProcessor) queueFor(userID string) chan messageTask {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(userID))
	return p.queues[int(hash.Sum32()%uint32(len(p.queues)))]
}

func (p *MessageProcessor) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	for _, queue := range p.queues {
		close(queue)
	}
}
