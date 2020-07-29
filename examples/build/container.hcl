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
}