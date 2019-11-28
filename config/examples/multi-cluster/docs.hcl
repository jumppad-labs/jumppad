docs "multi-cluster" {
  path  = "./docs"
  index = "index.html"
  port  = 8080
}

code "multi-cluster" {
  port    = 8080
  workdir = ""

  volume {
    source      = "./consul_config"
    destination = "/config"
  }
}
