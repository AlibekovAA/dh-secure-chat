package logger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSensitiveFieldsAreSanitized(t *testing.T) {
	directory := t.TempDir()
	log, err := New(directory, "test", "INFO")
	if err != nil {
		t.Fatal(err)
	}
	log.WithFields(context.Background(), Fields{
		"username": "alice", "client_ip": "127.0.0.1", "jti": "secret-jti", "user_id": "user-123", "action": "test",
	}).Info("message")
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(directory, "app.log"))
	if err != nil {
		t.Fatal(err)
	}
	output := string(contents)
	for _, forbidden := range []string{"alice", "127.0.0.1", "secret-jti", "user-123"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("sensitive value %q was logged", forbidden)
		}
	}
	if !strings.Contains(output, `"action":"test"`) || !strings.Contains(output, `"user_id":`) {
		t.Fatal("safe fields or pseudonymized user id missing")
	}
}
