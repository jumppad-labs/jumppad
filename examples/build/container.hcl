container "build" {
  build   {
    file = "./Dockerfile"
    context = "./src"
  }

  command = ["/bin/app"]

  port {
    local = 9090
    remote = 9090
    host = 9090
  }

  network {
    name = "network.onprem"
  }
}

network "onprem" {
  subnet = "10.6.0.0/16"
}