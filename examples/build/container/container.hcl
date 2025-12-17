variable "image" {
  default = ""
}

variable "network" {
  default = ""
}

variable "ingress_port" {
  default = 0
}

variable "default_port" {
  default = 19090
}

resource "random_number" "port" {
  minimum = 10000
  maximum = 20000
}

resource "container" "app" {
  image {
    name = variable.image
  }

  command = ["/bin/app"]

  port {
    local  = 9090
    remote = 9090
    host   = variable.ingress_port == 0 ? resource.random_number.port.value : variable.default_port
  }

  network {
    id = variable.network
  }
}

output "local_address" {
  value = "127.0.0.1"
}

output "local_port" {
  value = variable.ingress_port == 0 ? resource.random_number.port.value : variable.default_port
}