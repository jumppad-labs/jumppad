container_ingress "consul-http" {
  target  = "container.consul"

  port {
    local  = 8500
    remote = 8500
    host   = 18500
  }

  network  {
    name = "network.cloud"
  }
}

nomad_ingress "nomad-http" {
  cluster  = "nomad_cluster.dev"
  job = ""
  group = ""
  task = ""

  port {
    local  = 4646
    remote = 4646
    host   = 14646
    open_in_browser = true
  }

  network  {
    name = "network.cloud"
  }
}