resource "network" "main" {
  subnet = "10.10.0.0/16"
}

resource "container" "vault" {
  image {
    name = "vault:1.13.3"
  }

  network {
    id = resource.network.main.id
  }

  port {
    local = 8200
    remote = 8200
  }

  environment = {
    VAULT_DEV_ROOT_TOKEN_ID = "root"
  }

  privileged = true
}

resource "terraform" "configure_vault" {
  working_directory = "/terraform"

  environment = {
    VAULT_TOKEN = "root"
    VAULT_ADDR = "http://${resource.container.vault.container_name}:8200"
  }

  variables = {
    first = "first"
    second = 2
    third = {
      x = 3
      y = 3
    }
  }

  network {
    id = resource.network.main.id
  }

  volume {
    source = "${home()}/.terraform.d"
    destination = "/root/.terraform.d,ro"
  }

  volume {
    source = "workspace"
    destination = "/terraform"
  }
}

output "first" {
  value = resource.terraform.configure_vault.output.first
}