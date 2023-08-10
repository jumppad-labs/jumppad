resource "network" "dc1_enabled" {
  subnet = "10.15.0.0/16"
}

resource "container" "consul_enabled" {
  image {
    name = "consul:0.12.1"
  }

  command = ["consul", "agent", "-dev", "-client", "0.0.0.0"]

  network {
    id         = "abc"
    ip_address = "10.6.0.200"
  }

  port_range {
    range       = "8500-8502"
    enable_host = true
  }
}

resource "container" "consul_disabled" {
  disabled = true

  image {
    id   = "abc"
    name = "consul:0.12.1"
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