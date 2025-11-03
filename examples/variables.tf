variable "ygg_endpoint" {
  type        = string
  description = "Yggdrasil API endpoint"
}

variable "ygg_token" {
  type        = string
  sensitive   = true
  description = "Token for read operations"
}

variable "ygg_cred_key" {
  type        = string
  sensitive   = true
  description = "Credential key for write operations"
}

variable "ygg_cred_secret" {
  type        = string
  sensitive   = true
  description = "Credential secret for write operations"
}

variable "ca_cert_path" {
  type = string
}

variable "client_cert_path" {
  type = string
}

variable "client_key_path" {
  type = string
}