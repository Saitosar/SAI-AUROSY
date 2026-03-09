package secrets

import (
	"context"
	"os"
	"sync"
)

var (
	defaultProvider Provider
	defaultOnce    sync.Once
)

// Default returns the default secrets provider based on SECRETS_PROVIDER env.
// Values: env (default), vault, aws.
func Default(ctx context.Context) Provider {
	defaultOnce.Do(func() {
		switch os.Getenv("SECRETS_PROVIDER") {
		case "vault":
			if p, err := NewVaultProvider(); err == nil {
				defaultProvider = p
			} else {
				defaultProvider = NewEnvProvider()
			}
		case "aws":
			if p, err := NewAWSProvider(ctx); err == nil {
				defaultProvider = p
			} else {
				defaultProvider = NewEnvProvider()
			}
		default:
			defaultProvider = NewEnvProvider()
		}
	})
	return defaultProvider
}

// GetSecretOrEnv tries the provider first, then falls back to os.Getenv(key).
func GetSecretOrEnv(ctx context.Context, p Provider, key string) string {
	if p == nil {
		return os.Getenv(key)
	}
	v, err := p.GetSecret(ctx, key)
	if err != nil || v == "" {
		return os.Getenv(key)
	}
	return v
}
