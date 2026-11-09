package server

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

func TestShutdownClosesListenerDrainsRequestAndRunsHooks(t *testing.T) {
	log, err := logger.New(t.TempDir(), "test", "ERROR")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	entered := make(chan struct{})
	release := make(chan struct{})
	requestDone := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = server.Serve(listener) }()
	go func() {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr == nil {
			response.Body.Close()
		}
		close(requestDone)
	}()
	<-entered
	hookCalled := make(chan struct{})
	shutdownDone := make(chan struct{})
	go func() {
		shutdownServer(server, log, "test", []ShutdownHook{func(context.Context) error { close(hookCalled); return nil }})
		close(shutdownDone)
	}()
	select {
	case <-hookCalled:
	case <-time.After(time.Second):
		t.Fatal("shutdown hook not called")
	}
	if connection, dialErr := net.DialTimeout("tcp", listener.Addr().String(), 100*time.Millisecond); dialErr == nil {
		connection.Close()
		t.Fatal("listener still accepted connections")
	}
	close(release)
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("active request did not drain")
	}
	select {
	case <-shutdownDone:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not complete")
	}
}
