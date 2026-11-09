package websocket

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/clock"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type routeFunc func(context.Context, *Client, *WSMessage) error

func (fn routeFunc) Route(ctx context.Context, client *Client, message *WSMessage) error {
	return fn(ctx, client, message)
}

func testLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.New(t.TempDir(), "test", "ERROR")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	return log
}

func TestProcessorPreservesUserOrderAndCrossUserParallelism(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})
	otherStarted := make(chan struct{})
	router := routeFunc(func(_ context.Context, client *Client, message *WSMessage) error {
		switch string(message.Payload) {
		case "first":
			close(firstStarted)
			<-releaseFirst
		case "second":
			close(secondStarted)
		case "other":
			close(otherStarted)
		}
		return nil
	})
	processor := NewMessageProcessor(2, router, testLogger(t), 8)
	defer processor.Shutdown()
	first := &Client{userID: "user-a"}
	other := &Client{userID: "user-b"}
	for processor.queueFor(first.userID) == processor.queueFor(other.userID) {
		other.userID += "x"
	}

	if err := processor.Submit(context.Background(), first, &WSMessage{Payload: []byte("first")}); err != nil {
		t.Fatal(err)
	}
	<-firstStarted
	if err := processor.Submit(context.Background(), first, &WSMessage{Payload: []byte("second")}); err != nil {
		t.Fatal(err)
	}
	if err := processor.Submit(context.Background(), other, &WSMessage{Payload: []byte("other")}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-otherStarted:
	case <-time.After(time.Second):
		t.Fatal("other user did not run in parallel")
	}
	select {
	case <-secondStarted:
		t.Fatal("same-user task ran out of order")
	default:
	}
	close(releaseFirst)
	select {
	case <-secondStarted:
	case <-time.After(time.Second):
		t.Fatal("second task was not processed")
	}
}

func TestProcessorContinuesAfterTaskPanic(t *testing.T) {
	processed := make(chan struct{})
	var calls atomic.Int32
	processor := NewMessageProcessor(1, routeFunc(func(context.Context, *Client, *WSMessage) error {
		if calls.Add(1) == 1 {
			panic("boom")
		}
		close(processed)
		return nil
	}), testLogger(t), 2)
	defer processor.Shutdown()
	client := &Client{userID: "user"}
	_ = processor.Submit(context.Background(), client, &WSMessage{})
	_ = processor.Submit(context.Background(), client, &WSMessage{})
	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("worker stopped after panic")
	}
}

func TestIdempotencyExecutesConcurrentOperationOnce(t *testing.T) {
	tracker := NewIdempotencyTracker(context.Background(), time.Minute, clock.NewRealClock())
	defer tracker.Shutdown()
	var calls atomic.Int32
	start := make(chan struct{})
	entered := make(chan struct{})
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = tracker.Execute("operation", TypeMessage, func() (interface{}, error) {
				calls.Add(1)
				close(entered)
				<-start
				return "ok", nil
			})
		}()
	}
	<-entered
	close(start)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("operation executed %d times", calls.Load())
	}
}

func TestIdempotencyDoesNotCacheQueueRejectionAndReplaysErrors(t *testing.T) {
	tracker := NewIdempotencyTracker(context.Background(), time.Minute, clock.NewRealClock())
	defer tracker.Shutdown()
	if _, err := tracker.Execute("retry", TypeMessage, func() (interface{}, error) { return nil, ErrProcessorQueueFull }); !errors.Is(err, ErrProcessorQueueFull) {
		t.Fatalf("expected queue error, got %v", err)
	}
	if result, err := tracker.Execute("retry", TypeMessage, func() (interface{}, error) { return "accepted", nil }); err != nil || result != "accepted" {
		t.Fatalf("retry was not accepted: result=%v err=%v", result, err)
	}
	handlerErr := errors.New("handler failed")
	_, _ = tracker.Execute("error", TypeMessage, func() (interface{}, error) { return nil, handlerErr })
	if _, err := tracker.Execute("error", TypeMessage, func() (interface{}, error) { t.Fatal("cached error executed again"); return nil, nil }); !errors.Is(err, handlerErr) {
		t.Fatalf("expected cached handler error, got %v", err)
	}
}

func TestOldUnregisterDoesNotRemoveReplacement(t *testing.T) {
	hub := NewHub(HubDeps{Log: testLogger(t), Clock: clock.NewRealClock()}, HubConfig{MaxConnections: 2, SendTimeout: time.Second})
	oldClient := &Client{userID: "user", ctx: context.Background(), cancel: func() {}}
	newClient := &Client{userID: "user", ctx: context.Background(), cancel: func() {}}
	hub.clients.Store("user", newClient)
	hub.clientCount.Store(1)
	hub.handleUnregister(oldClient)
	stored, ok := hub.clients.Load("user")
	if !ok || stored.(*Client) != newClient || hub.clientCount.Load() != 1 {
		t.Fatal("old unregister removed replacement client")
	}
}
