package cognitive

import (
	"context"
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
