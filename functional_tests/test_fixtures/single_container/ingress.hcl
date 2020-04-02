container_ingress "consul-http" {
  target  = "container.consul"

  network {
    name = "network.onprem"
  }

  port {
    local  = 8500
    remote = 8500
    host   = 18500
  }
}