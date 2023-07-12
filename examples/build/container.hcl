resource "container" "app" {
  image {
    name = resource.build.app.image
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

resource "build" "app" {
  container {
    dockerfile = "Dockerfile"
    context    = "./src"
  }
}

resource "network" "onprem" {
  subnet = "10.6.0.0/16"
}