container "consul" {
  image   = "consul:1.6.1"
  command = ["consul", "agent", "-config-file=/config/consul.hcl"]

  volume {
    source      = "./consul_config"
    destination = "/config"
  }

  network    = "network.onprem"
  ip_address = "10.5.0.2" // optional
}

// consul.consul.ingress.local
ingress "consul" {
  target = "container.consul"

  port {
    local  = 8500
    remote = 8500
    host   = 18500
  }

  ip_address = "192.169.7.2" // if ommited will auto assign
}
