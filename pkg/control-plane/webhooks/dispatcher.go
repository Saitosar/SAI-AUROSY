package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Payload is the webhook event payload sent to external URLs.
type Payload struct {
	Event     string         `json:"event"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// Dispatcher sends webhook events to configured URLs.
type Dispatcher struct {
	store    Store
	client   *http.Client
	retries  int
	interval time.Duration
}

// NewDispatcher creates a new webhook dispatcher.
func NewDispatcher(store Store) *Dispatcher {
	return &Dispatcher{
		store:    store,
		client:   &http.Client{Timeout: 10 * time.Second},
		retries:  3,
		interval: 2 * time.Second,
	}
}

// Dispatch sends an event to all webhooks subscribed to it.
func (d *Dispatcher) Dispatch(ctx context.Context, event string, data map[string]any) {
	if d.store == nil {
		return
	}
	webhooks, err := d.store.ListByEvent(ctx, event)
	if err != nil {
		log.Printf("[webhooks] list by event %s: %v", event, err)
		return
	}
	payload := Payload{
		Event:     event,
		Timestamp: time.Now(),
		Data:     data,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[webhooks] marshal payload: %v", err)
		return
	}
	for _, wh := range webhooks {
		wh := wh
		go d.sendWithRetry(wh, body, payload.Event)
	}
}

func (d *Dispatcher) sendWithRetry(wh *Webhook, body []byte, event string) {
	for attempt := 0; attempt <= d.retries; attempt++ {
		if attempt > 0 {
			time.Sleep(d.interval)
		}
		if err := d.send(wh, body, event); err != nil {
			log.Printf("[webhooks] %s -> %s (attempt %d): %v", event, wh.URL, attempt+1, err)
			continue
		}
		return
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
