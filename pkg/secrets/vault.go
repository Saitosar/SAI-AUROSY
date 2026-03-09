package secrets

import (
	"context"
	"fmt"
	"os"
	"sync"

	vault "github.com/hashicorp/vault/api"
)

// VaultProvider reads secrets from HashiCorp Vault KV v2.
type VaultProvider struct {
	client *vault.Client
	path   string
	cache  map[string]string
	mu     sync.RWMutex
}

// NewVaultProvider creates a VaultProvider. Uses VAULT_ADDR, VAULT_TOKEN, VAULT_SECRET_PATH from env.
func NewVaultProvider() (*VaultProvider, error) {
	config := vault.DefaultConfig()
	if addr := os.Getenv("VAULT_ADDR"); addr != "" {
		if err := config.ReadEnvironment(); err != nil {
			return nil, fmt.Errorf("vault config: %w", err)
		}
	}
	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("vault client: %w", err)
	}
	token := os.Getenv("VAULT_TOKEN")
	if token != "" {
		client.SetToken(token)
	}
	secretPath := os.Getenv("VAULT_SECRET_PATH")
	if secretPath == "" {
		secretPath = "sai-aurosy"
	}
	return &VaultProvider{
		client: client,
		path:   secretPath,
		cache:  make(map[string]string),
	}, nil
}

// GetSecret fetches a secret from Vault KV v2. Path is the secret path, key is the field name.
func (p *VaultProvider) GetSecret(ctx context.Context, key string) (string, error) {
	p.mu.RLock()
	if v, ok := p.cache[key]; ok {
		p.mu.RUnlock()
		return v, nil
	}
	p.mu.RUnlock()

	secret, err := p.client.KVv2("secret").Get(ctx, p.path)
	if err != nil {
		return "", fmt.Errorf("vault get %s: %w", p.path, err)
	}
	if secret == nil || secret.Data == nil {
		return "", nil
	}
	val, _ := secret.Data[key].(string)
	if val != "" {
		p.mu.Lock()
		p.cache[key] = val
		p.mu.Unlock()
	}
	return val, nil
}
