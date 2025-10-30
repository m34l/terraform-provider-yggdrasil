resource "yggdrasil_secret" "db_password" {
  namespace = "example_namespace"
  key       = "example_key"
  value     = "example_value"
  tags = {
    app   = "example_tags"
    env   = "staging"
    owner = "cloud-foundation"
  }
}
