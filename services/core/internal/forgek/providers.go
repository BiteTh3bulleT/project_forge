package forgek

import (
	"fmt"
	"sync"
	"time"
)

type IDProvider interface {
	NextID(prefix string) string
}

type SequenceIDProvider struct {
	mu       sync.Mutex
	counters map[string]int
}

func NewSequenceIDProvider(counters map[string]int) *SequenceIDProvider {
	copyCounters := make(map[string]int, len(counters))
	for key, value := range counters {
		copyCounters[key] = value
	}
	return &SequenceIDProvider{counters: copyCounters}
}

func (p *SequenceIDProvider) NextID(prefix string) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.counters[prefix]++
	return fmt.Sprintf("%s-%04d", prefix, p.counters[prefix])
}

type Clock interface {
	Now() time.Time
}

type FixedClock struct {
	now time.Time
}

func NewFixedClock(now time.Time) FixedClock {
	return FixedClock{now: now}
}

func (c FixedClock) Now() time.Time {
	return c.now
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now().UTC()
}
