resource "network" "dc1_enabled" {
  subnet = "10.15.0.0/16"
}

resource "container" "consul_enabled" {
  image {
    name = "consul:1.10.6"
  }

  command = ["consul", "agent", "-dev", "-client", "0.0.0.0"]

  network {
    id         = resource.network.dc1_enabled.meta.id
    ip_address = "10.15.0.200"
  }

  port_range {
    range       = "8500-8502"
    enable_host = true
  }
}

resource "container" "consul_disabled" {
  disabled = true

  image {
    id   = resource.network.dc1_enabled.meta.id
    name = "consul:1.10.6"
  }

  command = ["consul", "agent", "-dev", "-client", "0.0.0.0"]

  network {
    ip_address = "10.6.0.200"
  }

  port_range {
    range       = "8500-8502"
    enable_host = true
  }
}