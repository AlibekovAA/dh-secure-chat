package clock

import "time"

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func NewRealClock() Clock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now()
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

func (c *MockClock) SetTime(t time.Time) {
	c.time = t
}
