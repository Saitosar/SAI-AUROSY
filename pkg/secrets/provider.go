package secrets

import "context"

// Provider resolves secrets from external stores (Vault, AWS Secrets Manager) or environment.
type Provider interface {
	GetSecret(ctx context.Context, key string) (string, error)
}
