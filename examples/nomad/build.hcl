resource "container" "registry" {
  image {
    name = "registry:2"
  }

  port {
    local = 5000
    host  = 5000
  }

  environment = {
    DEBUG = "true"
  }
}

resource "build" "app" {
  container {
    dockerfile = "./Docker/Dockerfile"
    context    = "./src"
    ignore     = ["**/.terraform"]
  }

  registry {
    name = "${resource.container.registry.container_name}:5000/mine:v0.1.0"
  }
}