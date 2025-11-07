# Terraform Provider for Yggdrasil

Terraform provider for managing configurations in Yggdrasil Configuration Service.

## Features

- Read configurations using token authentication (data sources)
- Write configurations using key-secret credentials (resources)
- Tag-based configuration management (production, staging, etc.)
- mTLS support for secure communication
- No deletion by design - prevents accidental configuration loss

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.5
- [Go](https://golang.org/doc/install) >= 1.21 (for development)
- Yggdrasil API access with:
  - Token (for read operations)
  - Key-Secret credentials (for write operations)
  - mTLS certificates

## Installation

### Terraform Registry (Recommended)

```hcl
terraform {
  required_providers {
    yggdrasil = {
      source  = "m34l/yggdrasil"
      version = "~> 1.0"
    }
  }
}
```

### Local Development

```bash
# Clone repository
git clone https://github.com/m34l/terraform-provider-yggdrasil.git
cd terraform-provider-yggdrasil

# Build provider
make build

# Install locally
make install-local
```

## Usage

### Provider Configuration

#### Option 1: Using File Paths (Traditional)

```hcl
provider "yggdrasil" {
  endpoint             = "https://yggdrasil.api.example.com"
  token                = var.ygg_token
  cred_key             = var.ygg_cred_key
  cred_secret          = var.ygg_cred_secret
  ca_cert_path         = "certs/cacert.crt"
  client_cert_path     = "certs/client.crt"
  client_key_path      = "certs/client.key"
  insecure_skip_verify = false
}
```

#### Option 2: Using Base64 Encoded Certs (Recommended for CI/CD)

```hcl
provider "yggdrasil" {
  endpoint             = "https://yggdrasil.api.example.com"
  token                = var.ygg_token
  cred_key             = var.ygg_cred_key
  cred_secret          = var.ygg_cred_secret
  ca_cert              = var.ygg_ca_cert        # base64 encoded
  client_cert          = var.ygg_client_cert    # base64 encoded
  client_key           = var.ygg_client_key     # base64 encoded
  insecure_skip_verify = false
}
```

### Environment Variables

```bash
# Authentication
export YGG_ENDPOINT="https://yggdrasil.api.example.com"
export YGG_TOKEN="your-token"
export YGG_CRED_KEY="your-key"
export YGG_CRED_SECRET="your-secret"

# Certificates as Base64 (Option 1)
export YGG_CA_CERT=$(base64 -i certs/cacert.crt)
export YGG_CLIENT_CERT=$(base64 -i certs/client.crt)
export YGG_CLIENT_KEY=$(base64 -i certs/client.key)

# Or use file paths (Option 2)
export YGG_CA_CERT_PATH="certs/cacert.crt"
export YGG_CLIENT_CERT_PATH="certs/client.crt"
export YGG_CLIENT_KEY_PATH="certs/client.key"
```

### Converting Certificates to Base64

```bash
# Linux/macOS
base64 -i certs/cacert.crt > cacert.b64
base64 -i certs/client.crt > client_cert.b64
base64 -i certs/client.key > client_key.b64

# Or inline
export YGG_CA_CERT=$(cat certs/cacert.crt | base64)
export YGG_CLIENT_CERT=$(cat certs/client.crt | base64)
export YGG_CLIENT_KEY=$(cat certs/client.key | base64)

# Windows PowerShell
$content = Get-Content certs/cacert.crt -Raw
[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($content))
```

### Example: Create Secret

```hcl
resource "yggdrasil_secret" "db_password" {
  namespace = "my-service"
  key       = "DB_PASSWORD"
  value     = "super-secret-password"
  tag       = "production"
}
```

### Example: Read Secret

```hcl
data "yggdrasil_secret" "db_password" {
  namespace = "my-service"
  key       = "DB_PASSWORD"
  tag       = "production"
}

output "password" {
  value     = data.yggdrasil_secret.db_password.value
  sensitive = true
}
```

### Example: Multiple Environments

```hcl
# Production
resource "yggdrasil_secret" "db_prod" {
  namespace = "my-service"
  key       = "DB_PASSWORD"
  value     = "prod-password"
  tag       = "production"
}

# Staging
resource "yggdrasil_secret" "db_staging" {
  namespace = "my-service"
  key       = "DB_PASSWORD"
  value     = "staging-password"
  tag       = "staging"
}
```

## Important Notes

### Deletion Behavior

**WARNING: Secrets are NOT deleted from Yggdrasil when you run `terraform destroy`.**

This is intentional to:
- Prevent accidental deletion of shared configuration
- Avoid database lock contention issues
- Allow other tags/environments to continue using the same keys

When you destroy a resource, it is only removed from **Terraform state**, but remains in **Yggdrasil**.

To actually delete secrets:
- Use Yggdrasil UI (Nidhoggr)
- Call Yggdrasil API directly
- Use `curl` with key-secret credentials

### Authentication

This provider uses **two types of authentication**:

1. **Token** (for read operations - data sources)
   - Used by `data "yggdrasil_secret"`
   - Read-only access

2. **Key-Secret** (for write operations - resources)
   - Used by `resource "yggdrasil_secret"`
   - Create/Update access
   - Authentication via headers: `key` and `secret`

## Authentication Setup

### 1. Get Token (for reads)

Login to Yggdrasil UI and copy your token from user settings.

### 2. Get Key-Secret Credentials (for writes)

1. Login to Yggdrasil UI (Nidhoggr)
2. Go to **User Settings** > **Credentials**
3. Click **Create New Credential**
4. Copy **Key** and **Secret** (secret is shown only once!)
5. Store credentials securely (use environment variables or secret manager)

### 3. Setup mTLS Certificates

Get the following certificates from your infrastructure team:
- `cacert.crt` - CA certificate
- `client.crt` - Client certificate
- `client.key` - Client private key

## Provider Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `endpoint` | string | Yes | Yggdrasil API base URL |
| `token` | string | Yes* | Token for read operations |
| `cred_key` | string | Yes* | Key for write operations |
| `cred_secret` | string | Yes* | Secret for write operations |
| `ca_cert_path` | string | No | Path to CA certificate file |
| `ca_cert` | string | No | Base64 encoded CA certificate (alternative to ca_cert_path) |
| `client_cert_path` | string | No | Path to client certificate file |
| `client_cert` | string | No | Base64 encoded client certificate (alternative to client_cert_path) |
| `client_key_path` | string | No | Path to client private key file |
| `client_key` | string | No | Base64 encoded client private key (alternative to client_key_path) |
| `insecure_skip_verify` | bool | No | Skip TLS verification (dev only) |
| `namespace_default` | string | No | Default namespace |

**Note**: 
- Base64 encoded certs take priority over file paths
- Token required for data sources, key-secret required for resources

## Resources

### `yggdrasil_secret`

Manages a configuration secret in Yggdrasil.

#### Arguments

- `namespace` (string, required) - Namespace where secret is stored
- `key` (string, required) - Configuration key name
- `value` (string, required, sensitive) - Configuration value
- `tag` (string, required) - Tag for this configuration (e.g., "production", "staging")

#### Attributes

- `id` (string) - Resource ID in format `namespace/key@tag`
- `version` (number) - Configuration version
- `updated_at` (string) - Last update timestamp
- `tags` (map) - Computed tags metadata

## Data Sources

### `yggdrasil_secret`

Reads a configuration secret from Yggdrasil.

#### Arguments

- `namespace` (string, required) - Namespace where secret is stored
- `key` (string, required) - Configuration key name
- `tag` (string, required) - Tag to read from

#### Attributes

- `id` (string) - Data source ID
- `value` (string, sensitive) - Configuration value
- `version` (number) - Configuration version
- `updated_at` (string) - Last update timestamp

## Development

### Prerequisites

- Go 1.21+
- Terraform 1.5+
- Make

### Building

```bash
# Build provider binary
make build

# Run tests
make test

# Run acceptance tests (requires Yggdrasil access)
make acc

# Generate documentation
make docs
```

### Testing

```bash
# Set up test credentials
export YGG_ENDPOINT="https://yggdrasil.api.example.com"
export YGG_TOKEN="your-test-token"
export YGG_CRED_KEY="your-test-key"
export YGG_CRED_SECRET="your-test-secret"

# Run tests
cd examples
terraform init
terraform plan
terraform apply
```

## Troubleshooting

### Error: Unauthenticated (401)

**Cause**: Invalid or expired credentials

**Solution**:
1. Verify token/key-secret are correct
2. Check if credentials are active in Yggdrasil UI
3. Ensure user has permission to access namespace

### Error: Lock contention (409)

**Cause**: Multiple operations on same namespace

**Solution**: This should not happen with current implementation (NO-OP delete). If it does, wait and retry.

### Error: Secret not found (404)

**Cause**: Configuration doesn't exist

**Solution**:
1. Verify namespace name is correct
2. Check if key exists in Yggdrasil UI
3. Ensure tag is correct (production, staging, etc.)

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

Mozilla Public License 2.0

## Support

- **Issues**: https://github.com/m34l/terraform-provider-yggdrasil/issues
- **Documentation**: https://registry.terraform.io/providers/m34l/yggdrasil/latest/docs

## Links

- [Terraform Provider Development](https://developer.hashicorp.com/terraform/plugin/framework)
- [Yggdrasil Documentation](https://yggdrasil-docs.example.com)

---

Made with love by Ismail Muhammad Zeindy
