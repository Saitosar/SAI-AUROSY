package secrets

import (
	"context"
	"os"
)

// EnvProvider reads secrets from environment variables.
type EnvProvider struct{}

// NewEnvProvider creates an EnvProvider.
func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}

// GetSecret returns the value of the environment variable named by key.
func (p *EnvProvider) GetSecret(ctx context.Context, key string) (string, error) {
	return os.Getenv(key), nil
}
