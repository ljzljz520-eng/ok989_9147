package runtime

import (
	"sync"
	"time"
)

type Clock struct {
	mu      sync.Mutex
	current time.Time
	step    time.Duration
}

func NewClock(start time.Time, step time.Duration) *Clock {
	if step <= 0 {
		step = time.Millisecond
	}
	return &Clock{current: start, step: step}
}

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	value := c.current
	c.current = c.current.Add(c.step)
	return value
}

func (c *Clock) Peek() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.current }

func (c *Clock) Advance(amount time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if amount > 0 {
		c.current = c.current.Add(amount)
	}
	return c.current
}

func (c *Clock) Reset(value time.Time) { c.mu.Lock(); defer c.mu.Unlock(); c.current = value }
