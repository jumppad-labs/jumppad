variable "image" {
  default = ""
}

variable "network" {
  default = ""
}

resource "container" "app" {
  image {
    name = variable.image
  }

  command = ["/bin/app"]

  port {
    local  = 9090
    remote = 9090
    host   = 9090
  }

  network {
    id = variable.network
  }
}