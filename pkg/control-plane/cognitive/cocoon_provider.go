package cognitive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func init() {
	Register("cocoon", NewCocoonGateway)
}

const (
	intentSystemPrompt = `Extract intent from mall visitor text. Return JSON only, no other text: {"intent":"find_store|greeting|goodbye|unknown","parameters":{"store_name":"..."},"confidence":0.0-1.0}. Supported intents: find_store (store_name), greeting, goodbye. For unknown or unclear text use intent "unknown".`
	planSystemPrompt   = `Given task_type and context, return JSON only, no other text: {"steps":[{"action":"string","payload":{},"duration_sec":0}]}. Actions: navigate, greet, standby, return_to_base. Each step must have action; payload and duration_sec are optional.`
)

// chatCompletionRequest is the OpenAI-compatible request for Cocoon.
type chatCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []chatMessage   `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Stream      bool            `json:"stream"`
}

// chatMessage is a single message in the chat.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionResponse is the OpenAI-compatible response from Cocoon.
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// CocoonGateway routes UnderstandIntent and Plan to Cocoon LLM; other capabilities use MockGateway.
type CocoonGateway struct {
	client    *http.Client
	baseURL   string
	model     string
	maxTokens int
	mock      *MockGateway
}

// NewCocoonGateway creates a Cocoon gateway from config.
func NewCocoonGateway(cfg Config) (Gateway, error) {
	timeout := 30 * time.Second
	if cfg.Cocoon.TimeoutSec > 0 {
		timeout = time.Duration(cfg.Cocoon.TimeoutSec) * time.Second
	}
	maxTokens := 512
	if cfg.Cocoon.MaxTokens > 0 {
		maxTokens = cfg.Cocoon.MaxTokens
	}
	model := "Qwen/Qwen3-32B"
	if cfg.Cocoon.Model != "" {
		model = cfg.Cocoon.Model
	}
	baseURL := strings.TrimSuffix(cfg.Cocoon.ClientURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:10000"
	}
	return &CocoonGateway{
		client:    &http.Client{Timeout: timeout},
		baseURL:   baseURL,
		model:     model,
		maxTokens: maxTokens,
		mock:      NewMockGateway(),
	}, nil
}

// Navigate delegates to MockGateway.
func (c *CocoonGateway) Navigate(ctx context.Context, req NavigateRequest) (*NavigateResult, error) {
	return c.mock.Navigate(ctx, req)
}

// Recognize delegates to MockGateway.
func (c *CocoonGateway) Recognize(ctx context.Context, req RecognizeRequest) (*RecognizeResult, error) {
	return c.mock.Recognize(ctx, req)
}

// Plan calls Cocoon chat completions and parses JSON steps.
func (c *CocoonGateway) Plan(ctx context.Context, req PlanRequest) (*PlanResult, error) {
	ctxJSON, _ := json.Marshal(req.Context)
	userPrompt := fmt.Sprintf("task_type: %s\ncontext: %s", req.TaskType, string(ctxJSON))
	content, err := c.chatCompletion(ctx, planSystemPrompt, userPrompt)
	if err != nil {
		return &PlanResult{Steps: []PlanStep{}}, nil
	}
	var parsed struct {
		Steps []PlanStep `json:"steps"`
	}
	if err := parseJSONContent(content, &parsed); err != nil {
		return &PlanResult{Steps: []PlanStep{}}, nil
	}
	return &PlanResult{Steps: parsed.Steps}, nil
}

// Transcribe delegates to MockGateway.
func (c *CocoonGateway) Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResult, error) {
	return c.mock.Transcribe(ctx, req)
}

// Synthesize delegates to MockGateway.
func (c *CocoonGateway) Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResult, error) {
	return c.mock.Synthesize(ctx, req)
}

// Translate delegates to MockGateway.
func (c *CocoonGateway) Translate(ctx context.Context, req TranslateRequest) (*TranslateResult, error) {
	return c.mock.Translate(ctx, req)
}

// UnderstandIntent calls Cocoon chat completions and parses JSON intent.
func (c *CocoonGateway) UnderstandIntent(ctx context.Context, req UnderstandIntentRequest) (*IntentResult, error) {
	if strings.TrimSpace(req.Text) == "" {
		return &IntentResult{Intent: "", Parameters: nil, Confidence: 0}, nil
	}
	content, err := c.chatCompletion(ctx, intentSystemPrompt, req.Text)
	if err != nil {
		return &IntentResult{Intent: "", Parameters: nil, Confidence: 0}, nil
	}
	var parsed IntentResult
	if err := parseJSONContent(content, &parsed); err != nil {
		return &IntentResult{Intent: "", Parameters: nil, Confidence: 0}, nil
	}
	return &parsed, nil
}

func (c *CocoonGateway) chatCompletion(ctx context.Context, systemPrompt, userContent string) (string, error) {
	body := chatCompletionRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Stream:    false,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cocoon HTTP %d", resp.StatusCode)
	}
	var res chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if len(res.Choices) == 0 || res.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty cocoon response")
	}
	return strings.TrimSpace(res.Choices[0].Message.Content), nil
}

// jsonBlockRegex extracts a JSON object from LLM output (handles markdown code blocks).
var jsonBlockRegex = regexp.MustCompile(`(?s)\{.*\}`)

func parseJSONContent(content string, v interface{}) error {
	content = strings.TrimSpace(content)
	if m := jsonBlockRegex.FindString(content); m != "" {
		content = m
	}
	return json.Unmarshal([]byte(content), v)
}
