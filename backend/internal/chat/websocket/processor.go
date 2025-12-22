package websocket

import (
	"context"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
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
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := p.router.Route(ctx, client, msg); err != nil {
		p.log.Warnf("websocket message processing failed user_id=%s type=%s: %v", client.userID, msg.Type, err)
	}
}

func (p *MessageProcessor) Submit(ctx context.Context, client *Client, msg *WSMessage) {
	task := messageTask{
		client: client,
		msg:    msg,
		ctx:    ctx,
	}

	select {
	case p.queue <- task:
	default:
		p.log.Warnf("websocket message queue full user_id=%s type=%s", client.userID, msg.Type)
	}
}

func (p *MessageProcessor) Shutdown() {
	close(p.queue)
}
