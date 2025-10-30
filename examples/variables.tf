variable "ygg_endpoint" {
  type = string
}

variable "ygg_token" {
  type      = string
  sensitive = true
}

variable "ca_cert_path" {
  type    = string
  default = ""
}

variable "client_cert_path" {
  type    = string
  default = ""
}

variable "client_key_path" {
  type    = string
  default = ""
}
