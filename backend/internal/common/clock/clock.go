package clock

import "time"

type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
}

type RealClock struct{}

func NewRealClock() Clock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now()
}

func (c *RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

type MockClock struct {
	time time.Time
}

func NewMockClock(t time.Time) *MockClock {
	return &MockClock{time: t}
}

func (c *MockClock) Now() time.Time {
	return c.time
}

func (c *MockClock) Since(t time.Time) time.Duration {
	return c.time.Sub(t)
}

func (c *MockClock) SetTime(t time.Time) {
	c.time = t
}
