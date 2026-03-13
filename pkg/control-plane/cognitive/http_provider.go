package cognitive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sai-aurosy/platform/pkg/secrets"
)

func init() {
	Register("http", NewHTTPGateway)
}

// HTTPGateway calls external AI services via REST.
type HTTPGateway struct {
	client             *http.Client
	navigateURL        string
	recognizeURL       string
	planURL            string
	transcribeURL       string
	synthesizeURL       string
	understandIntentURL string
	translateURL        string
	apiKey              string
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
		client:              &http.Client{Timeout: 30 * time.Second},
		navigateURL:         cfg.HTTP.NavigateURL,
		recognizeURL:        cfg.HTTP.RecognizeURL,
		planURL:             cfg.HTTP.PlanURL,
		transcribeURL:       cfg.HTTP.TranscribeURL,
		synthesizeURL:       cfg.HTTP.SynthesizeURL,
		understandIntentURL: cfg.HTTP.UnderstandIntentURL,
		translateURL:        cfg.HTTP.TranslateURL,
		apiKey:              apiKey,
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
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			var errBody struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(body, &errBody) == nil && errBody.Error != "" {
				return fmt.Errorf("%s", errBody.Error)
			}
		}
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

// Transcribe calls the configured STT service.
func (h *HTTPGateway) Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResult, error) {
	var res TranscribeResult
	if err := h.post(ctx, h.transcribeURL, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Synthesize calls the configured TTS service.
func (h *HTTPGateway) Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResult, error) {
	var res SynthesizeResult
	if err := h.post(ctx, h.synthesizeURL, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// UnderstandIntent calls the configured intent extraction service.
func (h *HTTPGateway) UnderstandIntent(ctx context.Context, req UnderstandIntentRequest) (*IntentResult, error) {
	var res IntentResult
	if err := h.post(ctx, h.understandIntentURL, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Translate calls the configured translation service.
func (h *HTTPGateway) Translate(ctx context.Context, req TranslateRequest) (*TranslateResult, error) {
	if h.translateURL == "" {
		return &TranslateResult{Text: req.Text}, nil
	}
	var res TranslateResult
	if err := h.post(ctx, h.translateURL, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
