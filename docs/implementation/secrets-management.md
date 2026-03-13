# Secrets Management

SAI AUROSY supports external secrets management via HashiCorp Vault or AWS Secrets Manager, avoiding storage of sensitive values (JWT_SECRET, DB credentials, API keys) in environment variables.

## Overview

| Provider | Env | Use Case |
|----------|-----|----------|
| `env` | Default | Development; reads from `os.Getenv` |
| `vault` | HashiCorp Vault | Production, on-prem |
| `aws` | AWS Secrets Manager | Production, AWS deployments |

## Configuration

Set `SECRETS_PROVIDER` to select the provider:

```bash
# Default: read from environment variables
export SECRETS_PROVIDER=env

# HashiCorp Vault
export SECRETS_PROVIDER=vault

# AWS Secrets Manager
export SECRETS_PROVIDER=aws
```

## Env Provider (Default)

No additional configuration. Secrets are read from environment variables:

- `JWT_SECRET`
- `JWT_PUBLIC_KEY`
- `REGISTRY_DB_DSN`
- `COGNITIVE_HTTP_API_KEY`
- `EDGE_API_KEY`
- `GEMINI_API_KEY` — used by the Gemini Adapter (see [Gemini Adapter](../integration/gemini-adapter.md))

## Vault Provider

### Environment Variables

| Variable | Description |
|----------|-------------|
| `VAULT_ADDR` | Vault server URL (e.g. `https://vault.example.com`) |
| `VAULT_TOKEN` | Vault token for authentication |
| `VAULT_SECRET_PATH` | Path to KV v2 secret (default: `sai-aurosy`) |

### Secret Structure

Store secrets in Vault KV v2 at the configured path. Keys must match the env var names:

```json
{
  "JWT_SECRET": "your-hmac-secret",
  "REGISTRY_DB_DSN": "postgres://user:pass@host:5432/db?sslmode=require",
  "COGNITIVE_HTTP_API_KEY": "optional-api-key",
  "EDGE_API_KEY": "sk-..."
}
```

### Example

```bash
# Write secrets to Vault
vault kv put secret/sai-aurosy \
  JWT_SECRET="$(openssl rand -base64 32)" \
  REGISTRY_DB_DSN="postgres://..."

# Configure Control Plane
export SECRETS_PROVIDER=vault
export VAULT_ADDR=https://vault.example.com
export VAULT_TOKEN=hvs.xxx
export VAULT_SECRET_PATH=sai-aurosy
```

## AWS Secrets Manager Provider

### Environment Variables

| Variable | Description |
|----------|-------------|
| `AWS_REGION` | AWS region for Secrets Manager |
| `AWS_SECRET_NAME` | Secret name or ARN (default: `sai-aurosy`) |

### Secret Structure

Store secrets as a JSON object in AWS Secrets Manager:

```json
{
  "JWT_SECRET": "your-hmac-secret",
  "REGISTRY_DB_DSN": "postgres://user:pass@host:5432/db?sslmode=require",
  "COGNITIVE_HTTP_API_KEY": "optional-api-key",
  "EDGE_API_KEY": "sk-..."
}
```

### Example

```bash
# Create secret in AWS
aws secretsmanager create-secret \
  --name sai-aurosy \
  --secret-string '{"JWT_SECRET":"...","REGISTRY_DB_DSN":"postgres://..."}'

# Configure Control Plane
export SECRETS_PROVIDER=aws
export AWS_REGION=us-east-1
export AWS_SECRET_NAME=sai-aurosy
```

## Fallback Behavior

`GetSecretOrEnv` tries the provider first, then falls back to `os.Getenv(key)`. This allows:

- Gradual migration: keep env vars while testing Vault/AWS
- Override: env var takes precedence if provider returns empty

## Migration from Environment

1. Set `SECRETS_PROVIDER=env` (default) — no change.
2. Create secrets in Vault or AWS with the same key names.
3. Set `SECRETS_PROVIDER=vault` or `aws` and configure provider env vars.
4. Remove sensitive env vars from deployment config.
5. Verify startup and auth.

## References

- [Phase 2.9 Security](phase-2.9-security.md)
- [Production Runbook](../operations/production-runbook.md)
- [Authentication](../integration/authentication.md)
