package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// AWSProvider reads secrets from AWS Secrets Manager.
type AWSProvider struct {
	client *secretsmanager.Client
	name   string
	cache  map[string]string
	mu     sync.RWMutex
}

// NewAWSProvider creates an AWSProvider. Uses AWS_REGION, AWS_SECRET_NAME from env.
func NewAWSProvider(ctx context.Context) (*AWSProvider, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	if region := os.Getenv("AWS_REGION"); region != "" {
		cfg.Region = region
	}
	name := os.Getenv("AWS_SECRET_NAME")
	if name == "" {
		name = "sai-aurosy"
	}
	return &AWSProvider{
		client: secretsmanager.NewFromConfig(cfg),
		name:   name,
		cache:  make(map[string]string),
	}, nil
}

// GetSecret fetches a secret from AWS Secrets Manager. Expects secret value to be JSON with key as field name.
func (p *AWSProvider) GetSecret(ctx context.Context, key string) (string, error) {
	p.mu.RLock()
	if v, ok := p.cache[key]; ok {
		p.mu.RUnlock()
		return v, nil
	}
	p.mu.RUnlock()

	out, err := p.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(p.name),
	})
	if err != nil {
		return "", fmt.Errorf("aws secretsmanager get: %w", err)
	}
	if out.SecretString == nil {
		return "", nil
	}
	var data map[string]string
	if err := json.Unmarshal([]byte(*out.SecretString), &data); err != nil {
		return "", fmt.Errorf("aws secret json: %w", err)
	}
	val := data[key]
	if val != "" {
		p.mu.Lock()
		p.cache[key] = val
		p.mu.Unlock()
	}
	return val, nil
}
