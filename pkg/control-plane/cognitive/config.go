package cognitive

import (
	"encoding/json"
	"fmt"
	"os"
)

// HTTPConfig holds HTTP provider URLs and auth.
type HTTPConfig struct {
	NavigateURL  string `json:"navigate_url,omitempty"`
	RecognizeURL string `json:"recognize_url,omitempty"`
	PlanURL     string `json:"plan_url,omitempty"`
	APIKeyEnv   string `json:"api_key_env,omitempty"` // env var name for API key
}

// Config holds Cognitive Gateway provider configuration.
type Config struct {
	Provider string     `json:"provider"`
	HTTP     HTTPConfig `json:"http,omitempty"`
}

// LoadConfig loads configuration from environment and optional config file.
// Env COGNITIVE_PROVIDER selects provider (mock, http). Default: mock.
// If COGNITIVE_CONFIG_PATH is set, JSON file overrides env.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Provider: "mock",
		HTTP: HTTPConfig{
			APIKeyEnv: "COGNITIVE_HTTP_API_KEY",
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
		if fileCfg.HTTP.APIKeyEnv != "" {
			cfg.HTTP.APIKeyEnv = fileCfg.HTTP.APIKeyEnv
		}
	}

	return cfg, nil
}
