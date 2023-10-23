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
    host  = 8200
  }

  environment = {
    VAULT_DEV_ROOT_TOKEN_ID = "root"
  }
}

resource "terraform" "configure_vault" {
  network {
    id = resource.network.main.id
  }

  environment = {
    VAULT_TOKEN = "root"
    VAULT_ADDR  = "http://${resource.container.vault.container_name}:8200"
  }

  variables = {
    first  = "one"
    second = 2
    third = {
      x = 3
      y = 4
    }
  }

  source            = "./workspace"
  working_directory = "/"
  version           = "1.6.2"
}

output "first" {
  value = resource.terraform.configure_vault.output.first
}

output "second" {
  value = resource.terraform.configure_vault.output.second
}

output "third_x" {
  value = resource.terraform.configure_vault.output.third.x
}

output "third_y" {
  value = resource.terraform.configure_vault.output.third.y
}

output "vault_secret" {
  value = resource.terraform.configure_vault.output.vault_secret
}