package cognitive

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// HTTPConfig holds HTTP provider URLs and auth.
type HTTPConfig struct {
	NavigateURL       string `json:"navigate_url,omitempty"`
	RecognizeURL      string `json:"recognize_url,omitempty"`
	PlanURL          string `json:"plan_url,omitempty"`
	TranscribeURL      string `json:"transcribe_url,omitempty"`
	SynthesizeURL      string `json:"synthesize_url,omitempty"`
	UnderstandIntentURL string `json:"understand_intent_url,omitempty"`
	TranslateURL       string `json:"translate_url,omitempty"`
	APIKeyEnv          string `json:"api_key_env,omitempty"` // env var name for API key
}

// CocoonConfig holds Cocoon provider configuration.
type CocoonConfig struct {
	ClientURL  string `json:"client_url,omitempty"`
	Model      string `json:"model,omitempty"`
	TimeoutSec int    `json:"timeout_sec,omitempty"`
	MaxTokens  int    `json:"max_tokens,omitempty"`
}

// Config holds Cognitive Gateway provider configuration.
type Config struct {
	Provider string       `json:"provider"`
	HTTP     HTTPConfig   `json:"http,omitempty"`
	Cocoon   CocoonConfig `json:"cocoon,omitempty"`
}

// LoadConfig loads configuration from environment and optional config file.
// Env COGNITIVE_PROVIDER selects provider (mock, http, cocoon). Default: mock.
// If COGNITIVE_CONFIG_PATH is set, JSON file overrides env.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Provider: "mock",
		HTTP: HTTPConfig{
			APIKeyEnv: "COGNITIVE_HTTP_API_KEY",
		},
		Cocoon: CocoonConfig{
			ClientURL:  "http://localhost:10000",
			Model:      "Qwen/Qwen3-32B",
			TimeoutSec: 30,
			MaxTokens:  512,
		},
	}

	// Env-based defaults
	if p := os.Getenv("COGNITIVE_PROVIDER"); p != "" {
		cfg.Provider = p
	}
	if u := os.Getenv("COGNITIVE_HTTP_NAV_URL"); u != "" {
		cfg.HTTP.NavigateURL = u
	}
	if u := os.Getenv("COGNITIVE_HTTP_RECOGNIZE_URL"); u != "" {
		cfg.HTTP.RecognizeURL = u
	}
	if u := os.Getenv("COGNITIVE_HTTP_PLAN_URL"); u != "" {
		cfg.HTTP.PlanURL = u
	}
	if u := os.Getenv("COGNITIVE_HTTP_TRANSCRIBE_URL"); u != "" {
		cfg.HTTP.TranscribeURL = u
	}
	if u := os.Getenv("COGNITIVE_HTTP_SYNTHESIZE_URL"); u != "" {
		cfg.HTTP.SynthesizeURL = u
	}
	if u := os.Getenv("COGNITIVE_HTTP_INTENT_URL"); u != "" {
		cfg.HTTP.UnderstandIntentURL = u
	}
	if u := os.Getenv("COGNITIVE_HTTP_TRANSLATE_URL"); u != "" {
		cfg.HTTP.TranslateURL = u
	}
	if u := os.Getenv("COGNITIVE_COCOON_CLIENT_URL"); u != "" {
		cfg.Cocoon.ClientURL = u
	}
	if m := os.Getenv("COGNITIVE_COCOON_MODEL"); m != "" {
		cfg.Cocoon.Model = m
	}
	if t := os.Getenv("COGNITIVE_COCOON_TIMEOUT_SEC"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			cfg.Cocoon.TimeoutSec = n
		}
	}
	if n := os.Getenv("COGNITIVE_COCOON_MAX_TOKENS"); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			cfg.Cocoon.MaxTokens = v
		}
	}

	// Config file overrides
	if path := os.Getenv("COGNITIVE_CONFIG_PATH"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cognitive config file: %w", err)
		}
		var fileCfg Config
		if err := json.Unmarshal(data, &fileCfg); err != nil {
			return nil, fmt.Errorf("cognitive config parse: %w", err)
		}
		if fileCfg.Provider != "" {
			cfg.Provider = fileCfg.Provider
		}
		if fileCfg.HTTP.NavigateURL != "" {
			cfg.HTTP.NavigateURL = fileCfg.HTTP.NavigateURL
		}
		if fileCfg.HTTP.RecognizeURL != "" {
			cfg.HTTP.RecognizeURL = fileCfg.HTTP.RecognizeURL
		}
		if fileCfg.HTTP.PlanURL != "" {
			cfg.HTTP.PlanURL = fileCfg.HTTP.PlanURL
		}
		if fileCfg.HTTP.TranscribeURL != "" {
			cfg.HTTP.TranscribeURL = fileCfg.HTTP.TranscribeURL
		}
		if fileCfg.HTTP.SynthesizeURL != "" {
			cfg.HTTP.SynthesizeURL = fileCfg.HTTP.SynthesizeURL
		}
		if fileCfg.HTTP.UnderstandIntentURL != "" {
			cfg.HTTP.UnderstandIntentURL = fileCfg.HTTP.UnderstandIntentURL
		}
		if fileCfg.HTTP.TranslateURL != "" {
			cfg.HTTP.TranslateURL = fileCfg.HTTP.TranslateURL
		}
		if fileCfg.HTTP.APIKeyEnv != "" {
			cfg.HTTP.APIKeyEnv = fileCfg.HTTP.APIKeyEnv
		}
		if fileCfg.Cocoon.ClientURL != "" {
			cfg.Cocoon.ClientURL = fileCfg.Cocoon.ClientURL
		}
		if fileCfg.Cocoon.Model != "" {
			cfg.Cocoon.Model = fileCfg.Cocoon.Model
		}
		if fileCfg.Cocoon.TimeoutSec > 0 {
			cfg.Cocoon.TimeoutSec = fileCfg.Cocoon.TimeoutSec
		}
		if fileCfg.Cocoon.MaxTokens > 0 {
			cfg.Cocoon.MaxTokens = fileCfg.Cocoon.MaxTokens
		}
	}

	return cfg, nil
}
