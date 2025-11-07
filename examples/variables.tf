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
  type        = string
  description = "Path to CA certificate file"
  default     = ""
}

variable "client_cert_path" {
  type        = string
  description = "Path to client certificate file"
  default     = ""
}

variable "client_key_path" {
  type        = string
  description = "Path to client private key file"
  default     = ""
}

# Base64 encoded certificates (NEW)
variable "ca_cert_base64" {
  type        = string
  sensitive   = true
  description = "Base64 encoded CA certificate"
  default     = ""
}

variable "client_cert_base64" {
  type        = string
  sensitive   = true
  description = "Base64 encoded client certificate"
  default     = ""
}

variable "client_key_base64" {
  type        = string
  sensitive   = true
  description = "Base64 encoded client private key"
  default     = ""
}