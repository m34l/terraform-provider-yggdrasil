# Example 1: Get value from specific tag 'production'
data "yggdrasil_secret" "db_password_prod" {
  namespace = "example_namespace"
  key       = "db_password"
  tag       = "production"
}

# Example 2: Get value from tag 'staging' - MISSING DECLARATION!
data "yggdrasil_secret" "db_password_staging" {
  namespace = "example_namespace"
  key       = "db_password"
  tag       = "staging"
}

# Example 3: Get another key from production
data "yggdrasil_secret" "api_key_prod" {
  namespace = "example_namespace"
  key       = "ISMAIL_GANTENG_BANGET"
  tag       = "production"
}

# Outputs
output "db_password_production" {
  value     = data.yggdrasil_secret.db_password_prod.value
  sensitive = true
}

output "db_password_staging" {
  value     = data.yggdrasil_secret.db_password_staging.value
  sensitive = true
}

output "secret_metadata" {
  value = {
    prod_namespace  = data.yggdrasil_secret.db_password_prod.namespace
    prod_key        = data.yggdrasil_secret.db_password_prod.key
    prod_version    = data.yggdrasil_secret.db_password_prod.version
    staging_version = data.yggdrasil_secret.db_password_staging.version
  }
}

# Debug output (BE CAREFULL!)
output "debug_values" {
  value = {
    production = nonsensitive(data.yggdrasil_secret.db_password_prod.value)
    staging    = nonsensitive(data.yggdrasil_secret.db_password_staging.value)
  }
  description = "WARNING: Plain text secrets!"
}
