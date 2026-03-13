package cognitive

import (
	"context"
	"regexp"
	"strings"
)

func init() {
	Register("mock", func(Config) (Gateway, error) { return NewMockGateway(), nil })
}

// MockGateway is a no-op gateway that returns empty or mock results.
// Use when no external AI provider is configured.
type MockGateway struct{}

// NewMockGateway returns a mock gateway.
func NewMockGateway() *MockGateway {
	return &MockGateway{}
}

// Navigate returns a simple straight-line path (mock).
func (m *MockGateway) Navigate(ctx context.Context, req NavigateRequest) (*NavigateResult, error) {
	path := [][]float64{}
	if len(req.From) >= 2 && len(req.To) >= 2 {
		path = [][]float64{req.From, req.To}
	}
	return &NavigateResult{
		Path:     path,
		Duration: 0,
		Distance: 0,
	}, nil
}

// Recognize returns empty objects (mock).
func (m *MockGateway) Recognize(ctx context.Context, req RecognizeRequest) (*RecognizeResult, error) {
	return &RecognizeResult{Objects: []RecognizedObject{}}, nil
}

// Plan returns empty steps (mock).
func (m *MockGateway) Plan(ctx context.Context, req PlanRequest) (*PlanResult, error) {
	return &PlanResult{Steps: []PlanStep{}}, nil
}

// Transcribe returns empty transcript (mock).
func (m *MockGateway) Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResult, error) {
	return &TranscribeResult{Text: "", Language: "en", Confidence: 0}, nil
}

// Synthesize returns empty audio (mock).
func (m *MockGateway) Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResult, error) {
	return &SynthesizeResult{AudioBase64: ""}, nil
}

// find_store keywords: store names and generic terms that map to find_store intent.
var findStoreKeywords = map[string]string{
	"nike": "Nike", "adidas": "Adidas",
	"store": "", "shop": "", "shops": "",
	"food": "Food Court", "restaurant": "Food Court", "restaurants": "Food Court",
	"electronics": "Electronics Zone",
}

// UnderstandIntent returns intent from keyword matching (mock). Maps store-related phrases to find_store.
func (m *MockGateway) UnderstandIntent(ctx context.Context, req UnderstandIntentRequest) (*IntentResult, error) {
	text := strings.ToLower(strings.TrimSpace(req.Text))
	if text == "" {
		return &IntentResult{Intent: "", Parameters: nil, Confidence: 0}, nil
	}
	// Extract potential store name: "where is nike" -> nike, "I want electronics" -> electronics
	storeName := extractStoreName(text)
	if storeName != "" {
		return &IntentResult{
			Intent:     "find_store",
			Parameters: map[string]interface{}{"store_name": storeName},
			Confidence: 0.9,
		}, nil
	}
	// Check keywords
	for kw, mapped := range findStoreKeywords {
		if strings.Contains(text, kw) {
			name := mapped
			if name == "" {
				name = kw
			}
			return &IntentResult{
				Intent:     "find_store",
				Parameters: map[string]interface{}{"store_name": name},
				Confidence: 0.8,
			}, nil
		}
	}
	return &IntentResult{Intent: "", Parameters: nil, Confidence: 0}, nil
}

// Translate returns text as-is (mock).
func (m *MockGateway) Translate(ctx context.Context, req TranslateRequest) (*TranslateResult, error) {
	return &TranslateResult{Text: req.Text}, nil
}

// extractStoreName tries to extract a store name from phrases like "where is X", "I want X", "find X".
var storeNamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)where\s+is\s+(\w+)`),
	regexp.MustCompile(`(?i)find\s+(\w+)`),
	regexp.MustCompile(`(?i)want\s+(\w+)`),
	regexp.MustCompile(`(?i)looking\s+for\s+(\w+)`),
	regexp.MustCompile(`(?i)need\s+(\w+)`),
}

func extractStoreName(text string) string {
	for _, re := range storeNamePatterns {
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			word := strings.ToLower(m[1])
			if mapped, ok := findStoreKeywords[word]; ok {
				if mapped != "" {
					return mapped
				}
				return capitalize(word)
			}
			return capitalize(word)
		}
	}
	return ""
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
