resource "container" "build" {
  build {
    dockerfile = "Dockerfile"
    context    = "./src"
  }

  command = ["/bin/app"]

  port {
    local  = 9090
    remote = 9090
    host   = 9090
  }

  network {
    id = resource.network.onprem.id
  }
}

resource "network" "onprem" {
  subnet = "10.6.0.0/16"
}