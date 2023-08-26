resource "network" "main" {
  subnet = "10.10.0.0/16"
}

resource "terraform" "test" {
  working_directory = "/terraform"

  environment = {
    TF_VAR_DEFAULT_FOLDER = "/terraform_basics"
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