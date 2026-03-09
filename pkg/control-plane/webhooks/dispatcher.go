package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/observability"
	"go.opentelemetry.io/otel/attribute"
)

// Payload is the webhook event payload sent to external URLs.
type Payload struct {
	Event     string         `json:"event"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// Dispatcher sends webhook events to configured URLs.
type Dispatcher struct {
	store         Store
	deadLetter    DeadLetterStore
	client        *http.Client
	retries       int
	circuitBreakers map[string]*CircuitBreaker
	cbMu          sync.RWMutex
}

// NewDispatcher creates a new webhook dispatcher.
func NewDispatcher(store Store) *Dispatcher {
	return NewDispatcherWithDeadLetter(store, nil)
}

// NewDispatcherWithDeadLetter creates a dispatcher with optional dead-letter store.
func NewDispatcherWithDeadLetter(store Store, deadLetter DeadLetterStore) *Dispatcher {
	return &Dispatcher{
		store:          store,
		deadLetter:     deadLetter,
		client:         &http.Client{Timeout: 10 * time.Second},
		retries:        3,
		circuitBreakers: make(map[string]*CircuitBreaker),
	}
}

// Dispatch sends an event to all webhooks subscribed to it.
func (d *Dispatcher) Dispatch(ctx context.Context, event string, data map[string]any) {
	if d.store == nil {
		return
	}
	ctx, end := observability.StartSpan(ctx, "webhook.dispatch",
		attribute.String("event", event),
	)
	defer end()

	webhooks, err := d.store.ListByEvent(ctx, event)
	if err != nil {
		slog.Error("webhooks list by event failed", "event", event, "error", err)
		return
	}
	payload := Payload{
		Event:     event,
		Timestamp: time.Now(),
		Data:     data,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("webhooks marshal payload failed", "error", err)
		return
	}
	for _, wh := range webhooks {
		wh := wh
		go d.sendWithRetry(ctx, wh, body, payload.Event)
	}
}

func (d *Dispatcher) getCircuitBreaker(url string) *CircuitBreaker {
	d.cbMu.RLock()
	cb, ok := d.circuitBreakers[url]
	d.cbMu.RUnlock()
	if ok {
		return cb
	}
	d.cbMu.Lock()
	defer d.cbMu.Unlock()
	if cb, ok = d.circuitBreakers[url]; ok {
		return cb
	}
	cb = NewCircuitBreaker()
	d.circuitBreakers[url] = cb
	return cb
}

func (d *Dispatcher) sendWithRetry(ctx context.Context, wh *Webhook, body []byte, event string) {
	cb := d.getCircuitBreaker(wh.URL)

	for attempt := 0; attempt <= d.retries; attempt++ {
		if !cb.Allow() {
			slog.Warn("webhooks circuit open, skipping", "url", wh.URL, "event", event)
			if d.deadLetter != nil {
				_ = d.deadLetter.Record(ctx, wh.ID, event, body, fmt.Errorf("circuit open"))
			}
			return
		}

		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			time.Sleep(backoff)
		}

		err := d.send(wh, body, event)
		if err != nil {
			cb.RecordFailure()
			slog.Warn("webhooks delivery failed", "event", event, "url", wh.URL, "attempt", attempt+1, "error", err)
			continue
		}
		cb.RecordSuccess()
		return
	}

	if d.deadLetter != nil {
		_ = d.deadLetter.Record(ctx, wh.ID, event, body, fmt.Errorf("all %d attempts failed", d.retries+1))
	}
}

func (d *Dispatcher) send(wh *Webhook, body []byte, event string) error {
	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", event)
	if wh.Secret != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret))
		mac.Write(body)
		req.Header.Set("X-Webhook-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
