package events

import (
	"encoding/json"
	"sync"
	"time"
)

// Event represents a broadcast event for SSE clients.
type Event struct {
	Type      string         `json:"event"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// Subscriber receives events. Send on the channel; closing the channel unsubscribes.
type Subscriber chan Event

// Broadcaster distributes events to connected SSE subscribers.
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[Subscriber]struct{}
}

// NewBroadcaster creates a new event broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[Subscriber]struct{}),
	}
}

// Subscribe adds a new subscriber. Caller must eventually call Unsubscribe.
func (b *Broadcaster) Subscribe() Subscriber {
	ch := make(Subscriber, 32)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber.
func (b *Broadcaster) Unsubscribe(ch Subscriber) {
	b.mu.Lock()
	delete(b.subscribers, ch)
	b.mu.Unlock()
	close(ch)
}

// Broadcast sends an event to all subscribers. Non-blocking; drops if subscriber buffer is full.
func (b *Broadcaster) Broadcast(eventType string, data map[string]any) {
	e := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
	if e.Data == nil {
		e.Data = make(map[string]any)
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers {
		select {
		case ch <- e:
		default:
			// Subscriber buffer full, skip
		}
	}
}

// EventJSON returns the event as JSON bytes for SSE data.
func (e Event) EventJSON() ([]byte, error) {
	return json.Marshal(e)
}
