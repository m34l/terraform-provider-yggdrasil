# Create secret for production environment
resource "yggdrasil_secret" "db_password_prod" {
  namespace = "example_namespace"
  key       = "DB_PASSWORD"
  value     = "super-secret-prod-password-123!"
  tag       = "production"
}

# Create secret for production environment
resource "yggdrasil_secret" "db_host_prod" {
  namespace = "example_namespace"
  key       = "DB_HOST"
  value     = "localhost"
  tag       = "staging"
}

# Create secret for staging environment
resource "yggdrasil_secret" "db_password_staging" {
  namespace = "example_namespace"
  key       = "DB_PASSWORD"
  value     = "staging-password-456"
  tag       = "staging"
}

# Create another secret
resource "yggdrasil_secret" "api_key_prod" {
  namespace = "example_namespace"
  key       = "API_KEY"
  value     = "sk-prod-1234567890abcdef"
  tag       = "production"
}

# Output resource ID
output "db_password_prod_id" {
  value = yggdrasil_secret.db_password_prod.id
}

output "db_password_prod_version" {
  value = yggdrasil_secret.db_password_prod.version
}
