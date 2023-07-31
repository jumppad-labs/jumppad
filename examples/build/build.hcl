variable "container_enabled" {
  default = true
}

variable "nomad_enabled" {
  default = true
}

resource "build" "app" {
  container {
    dockerfile = "Dockerfile"
    context    = "./src"
  }
}

resource "network" "onprem" {
  subnet = "10.6.0.0/16"
}

module "container" {
  disabled = !variable.container_enabled
  source   = "./container"

  variables = {
    image   = resource.build.app.image
    network = resource.network.onprem.id
  }
}

module "nomad" {
  disabled = !variable.nomad_enabled
  source   = "./nomad"

  variables = {
    image   = resource.build.app.image
    network = resource.network.onprem.id
  }
}