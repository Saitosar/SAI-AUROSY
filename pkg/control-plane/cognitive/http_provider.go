package cognitive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sai-aurosy/platform/pkg/secrets"
)

func init() {
	Register("http", NewHTTPGateway)
}

// HTTPGateway calls external AI services via REST.
type HTTPGateway struct {
	client       *http.Client
	navigateURL  string
	recognizeURL string
	planURL      string
	apiKey       string
}

// NewHTTPGateway creates an HTTP gateway from config.
func NewHTTPGateway(cfg Config) (Gateway, error) {
	apiKey := ""
	if cfg.HTTP.APIKeyEnv != "" {
		ctx := context.Background()
		p := secrets.Default(ctx)
		apiKey = secrets.GetSecretOrEnv(ctx, p, cfg.HTTP.APIKeyEnv)
	}
	return &HTTPGateway{
		client:       &http.Client{Timeout: 30 * time.Second},
		navigateURL:  cfg.HTTP.NavigateURL,
		recognizeURL: cfg.HTTP.RecognizeURL,
		planURL:      cfg.HTTP.PlanURL,
		apiKey:       apiKey,
	}, nil
}

func (h *HTTPGateway) post(ctx context.Context, url string, reqBody, resBody interface{}) error {
	if url == "" {
		return fmt.Errorf("URL not configured")
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if h.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.apiKey)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(resBody)
}

// Navigate calls the configured navigation service.
func (h *HTTPGateway) Navigate(ctx context.Context, req NavigateRequest) (*NavigateResult, error) {
	var res NavigateResult
	if err := h.post(ctx, h.navigateURL, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Recognize calls the configured recognition service.
func (h *HTTPGateway) Recognize(ctx context.Context, req RecognizeRequest) (*RecognizeResult, error) {
	var res RecognizeResult
	if err := h.post(ctx, h.recognizeURL, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Plan calls the configured planning service.
func (h *HTTPGateway) Plan(ctx context.Context, req PlanRequest) (*PlanResult, error) {
	var res PlanResult
	if err := h.post(ctx, h.planURL, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
