package config

import (
	"testing"
	"time"
)

func TestTypedEnvironmentParsing(t *testing.T) {
	t.Run("absent uses default", func(t *testing.T) {
		t.Setenv("TEST_DURATION", "")
		got, err := getDurationEnv("TEST_DURATION", time.Second)
		if err != nil || got != time.Second {
			t.Fatalf("got %v, %v", got, err)
		}
	})
	t.Run("valid", func(t *testing.T) {
		t.Setenv("TEST_DURATION", "2s")
		got, err := getDurationEnv("TEST_DURATION", time.Second)
		if err != nil || got != 2*time.Second {
			t.Fatalf("got %v, %v", got, err)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		t.Setenv("TEST_DURATION", "later")
		if _, err := getDurationEnv("TEST_DURATION", time.Second); err == nil {
			t.Fatal("expected error")
		}
	})
	for _, test := range []struct {
		value string
		want  int
	}{{"0", 0}, {"-1", -1}} {
		t.Run(test.value, func(t *testing.T) {
			t.Setenv("TEST_INT", test.value)
			got, err := getIntEnv("TEST_INT", 1)
			if err != nil || got != test.want {
				t.Fatalf("got %d, %v", got, err)
			}
		})
	}
}

func TestLoadAuthConfigRejectsInvalidPort(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-that-is-long-enough-12345")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	for _, port := range []string{"0", "65536", "invalid"} {
		t.Run(port, func(t *testing.T) {
			t.Setenv("AUTH_HTTP_PORT", port)
			if _, err := LoadAuthConfig(); err == nil {
				t.Fatal("expected invalid port error")
			}
		})
	}
}

func TestLoadAuthConfigRejectsNonPositiveValues(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-that-is-long-enough-12345")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	for _, value := range []string{"0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("AUTH_REQUEST_TIMEOUT", value)
			if _, err := LoadAuthConfig(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
