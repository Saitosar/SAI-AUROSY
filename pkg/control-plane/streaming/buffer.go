package streaming

import (
	"container/ring"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sai-aurosy/platform/pkg/hal"
)

const (
	// DefaultBufferCapacity is the default number of telemetry events to retain for reconnect.
	DefaultBufferCapacity = 1000
	// DefaultBackpressureCapacity is the channel capacity for per-stream backpressure (drop oldest when full).
	DefaultBackpressureCapacity = 1000
)

var droppedEventsTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "streaming_gateway_dropped_events_total",
		Help: "Total number of telemetry events dropped due to backpressure",
	},
)

func init() {
	prometheus.MustRegister(droppedEventsTotal)
}

// BufferedEvent holds a telemetry event with an ID for Last-Event-ID support.
type BufferedEvent struct {
	ID        string         // Timestamp-based ID for Last-Event-ID
	Timestamp time.Time      // Original timestamp
	Telemetry *hal.Telemetry // Event payload
}

// RingBuffer is a thread-safe ring buffer for telemetry events.
type RingBuffer struct {
	mu       sync.RWMutex
	ring     *ring.Ring
	capacity int
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = DefaultBufferCapacity
	}
	return &RingBuffer{
		ring:     ring.New(capacity),
		capacity: capacity,
	}
}

// Add appends a telemetry event to the buffer (drops oldest when full).
func (b *RingBuffer) Add(t *hal.Telemetry) {
	id := eventID(t.Timestamp)
	ev := &BufferedEvent{ID: id, Timestamp: t.Timestamp, Telemetry: t}
	b.mu.Lock()
	b.ring.Value = ev
	b.ring = b.ring.Next()
	b.mu.Unlock()
}

// GetSince returns events with ID greater than lastID (for reconnect), sorted by ID.
// lastID is typically a timestamp in RFC3339Nano format.
func (b *RingBuffer) GetSince(lastID string) []*BufferedEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var out []*BufferedEvent
	b.ring.Do(func(v interface{}) {
		if v == nil {
			return
		}
		ev := v.(*BufferedEvent)
		if ev.ID > lastID {
			out = append(out, ev)
		}
	})
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// eventID generates an ID from timestamp for Last-Event-ID (lexicographically sortable).
func eventID(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// IncDropped increments the dropped events counter (call when backpressure drops an event).
func IncDropped() {
	droppedEventsTotal.Inc()
}
