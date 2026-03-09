package webhooks

import (
	"sync"
	"time"
)

// CircuitBreaker prevents cascading failures when a webhook endpoint is down.
// States: closed (normal), open (reject), half-open (probe).
type CircuitBreaker struct {
	mu sync.RWMutex

	failureThreshold int
	openDuration     time.Duration

	failures   int
	lastFail   time.Time
	state      state
}

type state int

const (
	stateClosed state = iota
	stateOpen
	stateHalfOpen
)

// NewCircuitBreaker creates a circuit breaker with FailureThreshold=5, OpenDuration=60s.
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: 5,
		openDuration:     60 * time.Second,
		state:            stateClosed,
	}
}

// Allow returns true if the request should be sent.
func (c *CircuitBreaker) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	switch c.state {
	case stateClosed:
		return true
	case stateOpen:
		if now.Sub(c.lastFail) >= c.openDuration {
			c.state = stateHalfOpen
			return true
		}
		return false
	case stateHalfOpen:
		return true
	}
	return true
}

// RecordSuccess resets the circuit on success.
func (c *CircuitBreaker) RecordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = 0
	c.state = stateClosed
}

// RecordFailure records a failure and may open the circuit.
func (c *CircuitBreaker) RecordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures++
	c.lastFail = time.Now()
	if c.state == stateHalfOpen {
		c.state = stateOpen
		return
	}
	if c.failures >= c.failureThreshold {
		c.state = stateOpen
	}
}
