terraform {
  required_providers {
    vault = {
      source = "hashicorp/vault"
      version = "3.19.0"
    }
  }
}

provider "vault" {}

resource "vault_generic_secret" "example" {
  path = "secret/foo"
  data_json = <<-EOF
  {
    "foo":   "bar",
    "pizza": "cheese"
  }
  EOF
}