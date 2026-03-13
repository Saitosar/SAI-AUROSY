package cognitive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCocoonGateway_UnderstandIntent(t *testing.T) {
	response := chatCompletionResponse{
		Choices: []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{
			{Message: struct {
				Content string `json:"content"`
			}{Content: `{"intent":"find_store","parameters":{"store_name":"Nike"},"confidence":0.95}`}},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	cfg := Config{
		Provider: "cocoon",
		Cocoon: CocoonConfig{
			ClientURL:  srv.URL,
			Model:      "Qwen/Qwen3-8B",
			TimeoutSec: 5,
			MaxTokens:  256,
		},
	}
	gw, err := NewCocoonGateway(cfg)
	if err != nil {
		t.Fatalf("NewCocoonGateway: %v", err)
	}

	res, err := gw.UnderstandIntent(context.Background(), UnderstandIntentRequest{
		RobotID:  "r1",
		Text:     "Where is Nike?",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("UnderstandIntent: %v", err)
	}
	if res.Intent != "find_store" {
		t.Errorf("Intent = %q, want find_store", res.Intent)
	}
	if res.Confidence != 0.95 {
		t.Errorf("Confidence = %v, want 0.95", res.Confidence)
	}
	if sn, ok := res.Parameters["store_name"].(string); !ok || sn != "Nike" {
		t.Errorf("Parameters[store_name] = %v, want Nike", res.Parameters["store_name"])
	}
}

func TestCocoonGateway_UnderstandIntent_EmptyText(t *testing.T) {
	cfg := Config{Provider: "cocoon", Cocoon: CocoonConfig{ClientURL: "http://invalid"}}
	gw, err := NewCocoonGateway(cfg)
	if err != nil {
		t.Fatalf("NewCocoonGateway: %v", err)
	}
	res, err := gw.UnderstandIntent(context.Background(), UnderstandIntentRequest{Text: ""})
	if err != nil {
		t.Fatalf("UnderstandIntent: %v", err)
	}
	if res.Intent != "" || res.Confidence != 0 {
		t.Errorf("expected empty intent, got %q confidence %v", res.Intent, res.Confidence)
	}
}

func TestCocoonGateway_UnderstandIntent_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := Config{Provider: "cocoon", Cocoon: CocoonConfig{ClientURL: srv.URL}}
	gw, err := NewCocoonGateway(cfg)
	if err != nil {
		t.Fatalf("NewCocoonGateway: %v", err)
	}
	res, err := gw.UnderstandIntent(context.Background(), UnderstandIntentRequest{Text: "hello"})
	if err != nil {
		t.Fatalf("UnderstandIntent: %v", err)
	}
	if res.Intent != "" {
		t.Errorf("expected empty intent on HTTP error, got %q", res.Intent)
	}
}

func TestCocoonGateway_Plan(t *testing.T) {
	response := chatCompletionResponse{
		Choices: []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{
			{Message: struct {
				Content string `json:"content"`
			}{Content: `{"steps":[{"action":"navigate","payload":{"target":"store-nike"},"duration_sec":60},{"action":"greet","duration_sec":5}]}`}},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	cfg := Config{Provider: "cocoon", Cocoon: CocoonConfig{ClientURL: srv.URL}}
	gw, err := NewCocoonGateway(cfg)
	if err != nil {
		t.Fatalf("NewCocoonGateway: %v", err)
	}

	res, err := gw.Plan(context.Background(), PlanRequest{
		TaskType: "mall_assistant",
		Context:  map[string]interface{}{"store_name": "Nike"},
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(res.Steps) != 2 {
		t.Fatalf("Steps length = %d, want 2", len(res.Steps))
	}
	if res.Steps[0].Action != "navigate" {
		t.Errorf("Steps[0].Action = %q, want navigate", res.Steps[0].Action)
	}
	if res.Steps[1].Action != "greet" {
		t.Errorf("Steps[1].Action = %q, want greet", res.Steps[1].Action)
	}
}

func TestCocoonGateway_MockDelegation(t *testing.T) {
	cfg := Config{Provider: "cocoon", Cocoon: CocoonConfig{ClientURL: "http://invalid"}}
	gw, err := NewCocoonGateway(cfg)
	if err != nil {
		t.Fatalf("NewCocoonGateway: %v", err)
	}

	nav, _ := gw.Navigate(context.Background(), NavigateRequest{From: []float64{0, 0}, To: []float64{1, 1}})
	if len(nav.Path) != 2 {
		t.Errorf("Navigate: expected 2 waypoints, got %d", len(nav.Path))
	}

	rec, _ := gw.Recognize(context.Background(), RecognizeRequest{})
	if rec.Objects != nil && len(rec.Objects) != 0 {
		t.Errorf("Recognize: expected empty objects, got %d", len(rec.Objects))
	}

	tr, _ := gw.Transcribe(context.Background(), TranscribeRequest{})
	if tr.Text != "" {
		t.Errorf("Transcribe: expected empty, got %q", tr.Text)
	}

	syn, _ := gw.Synthesize(context.Background(), SynthesizeRequest{Text: "hi", Language: "en"})
	if syn.AudioBase64 != "" {
		t.Errorf("Synthesize: expected empty, got non-empty")
	}
}

func TestParseJSONContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"plain json", `{"intent":"x"}`, false},
		{"with markdown", "```json\n{\"intent\":\"x\"}\n```", false},
		{"invalid", "not json", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v struct {
				Intent string `json:"intent"`
			}
			err := parseJSONContent(tt.content, &v)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSONContent() err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && v.Intent != "x" {
				t.Errorf("Intent = %q, want x", v.Intent)
			}
		})
	}
}
