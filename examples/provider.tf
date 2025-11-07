terraform {
  required_version = ">= 1.5.0"

  required_providers {
    yggdrasil = {
      source  = "m34l/yggdrasil"
      version = ">= 0.0.1"
    }
  }
}

provider "yggdrasil" {
  endpoint             = var.ygg_endpoint
  token                = var.ygg_token        # For read operations (data sources)
  cred_key             = var.ygg_cred_key     # For write operations (resources)
  cred_secret          = var.ygg_cred_secret  # For write operations (resources)
  namespace_default    = "internal-tooling"
  ca_cert_path         = var.ca_cert_path
  client_cert_path     = var.client_cert_path
  client_key_path      = var.client_key_path
  ca_cert              = var.ca_cert_base64
  client_cert          = var.client_cert_base64
  client_key           = var.client_key_base64
  insecure_skip_verify = false
}
