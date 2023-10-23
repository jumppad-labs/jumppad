terraform {
  required_providers {
    vault = {
      source = "hashicorp/vault"
      version = "3.19.0"
    }
  }
}

provider "vault" {}

resource "vault_mount" "kvv1" {
  path        = "kvv1"
  type        = "kv"
  options     = { version = "1" }
  description = "KV Version 1 secret engine mount"
}

resource "vault_kv_secret" "secret" {
  path = "${vault_mount.kvv1.path}/secret"
  data_json = jsonencode(
  {
    zip = "zap",
    foo = "bar"
  }
  )
}

data "vault_kv_secret" "secret_data" {
  path = vault_kv_secret.secret.path
}

output "vault_secret" {
  sensitive = true
  value = data.vault_kv_secret.secret_data.data.zip 
}