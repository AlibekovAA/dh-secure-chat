package websocket

import (
	"sync"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type MessageBatcher struct {
	messages      []*WSMessage
	timer         *time.Timer
	mu            sync.Mutex
	handler       func([]*WSMessage)
	log           *logger.Logger
	batchSize     int
	flushInterval time.Duration
}

func NewMessageBatcher(handler func([]*WSMessage), log *logger.Logger, batchSize int, flushInterval time.Duration) *MessageBatcher {
	if batchSize <= 0 {
		batchSize = 10
	}
	if flushInterval <= 0 {
		flushInterval = 100 * time.Millisecond
	}

	return &MessageBatcher{
		messages:      make([]*WSMessage, 0, batchSize),
		handler:       handler,
		log:           log,
		batchSize:     batchSize,
		flushInterval: flushInterval,
	}
}

func (b *MessageBatcher) Add(msg *WSMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.messages = append(b.messages, msg)

	if len(b.messages) >= b.batchSize {
		b.flush()
	} else if b.timer == nil {
		b.timer = time.AfterFunc(b.flushInterval, b.flush)
	}
}

func (b *MessageBatcher) flush() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.messages) == 0 {
		if b.timer != nil {
			b.timer.Stop()
			b.timer = nil
		}
		return
	}

	messages := make([]*WSMessage, len(b.messages))
	copy(messages, b.messages)
	b.messages = b.messages[:0]

	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				b.log.Errorf("message batcher handler panic: %v", r)
			}
		}()
		b.handler(messages)
	}()
}

func (b *MessageBatcher) Shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	if len(b.messages) > 0 {
		b.flush()
	}
}
